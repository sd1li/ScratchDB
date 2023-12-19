package queuestorage

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"scratchdata/models"
	"scratchdata/pkg/database"
	"scratchdata/pkg/destinations"
	"scratchdata/pkg/filestore"
	"scratchdata/pkg/queue"
	"scratchdata/util"

	"github.com/rs/zerolog/log"
)

var DefaultWriterOptions = WriterOptions{
	DataDir:     "./data",
	MaxFileSize: 100 * 1024 * 1024, // 100MB
	MaxRows:     1_000,
	MaxFileAge:  1 * time.Hour,
}

type WriterOptions struct {
	DataDir     string
	MaxFileSize int64
	MaxRows     int64
	MaxFileAge  time.Duration
}

type QueueStorageParam struct {
	Queue   queue.QueueBackend
	Storage filestore.StorageBackend

	WriterOpt    WriterOptions // TODO: Refactor use of this
	TimeProvider func() time.Time

	DB      database.Database
	Workers int
	DataDir string
}

type QueueStorage struct {
	queue   queue.QueueBackend
	storage filestore.StorageBackend

	DB      database.Database
	DataDir string
	Workers int

	fws          map[string]*FileWriter
	fwsMu        sync.Mutex
	closedFiles  chan FileWriterInfo
	timeProvider func() time.Time

	wg   sync.WaitGroup
	done chan bool

	opt WriterOptions
}

func NewQueueStorageTransport(param QueueStorageParam) *QueueStorage {
	rc := &QueueStorage{
		queue:        param.Queue,
		storage:      param.Storage,
		timeProvider: param.TimeProvider,
		opt:          param.WriterOpt,

		fws:         make(map[string]*FileWriter),
		closedFiles: make(chan FileWriterInfo),

		DB:      param.DB,
		DataDir: param.DataDir,
		Workers: param.Workers,
	}

	return rc
}

func (s *QueueStorage) StartProducer() error {
	log.Info().Msg("Starting data producer")
	return nil
}

func (s *QueueStorage) StopProducer() error {
	log.Info().Msg("Stopping data producer")

	var err error
	s.fwsMu.Lock()
	defer s.fwsMu.Unlock()
	for k, v := range s.fws {
		if closeErr := v.Close(); closeErr != nil {
			log.Error().Err(closeErr).Msg("unable to close file")
			err = errors.Join(err, closeErr)
		}
		delete(s.fws, k)
	}

	return err
}

func (s *QueueStorage) Write(databaseConnectionId string, table string, data []byte) (err error) {
	s.fwsMu.Lock()
	defer s.fwsMu.Unlock()
	fw, ok := s.fws[databaseConnectionId]
	if !ok {
		var err error
		fw, err = NewFileWriter(FileWriterParam{
			Key:         databaseConnectionId,
			Dir:         s.opt.DataDir,
			MaxFileSize: s.opt.MaxFileSize,
			MaxRows:     s.opt.MaxRows,
			MaxFileAge:  s.opt.MaxFileAge,

			Queue:   s.queue,
			Storage: s.storage,
		})
		if err != nil {
			return err
		}
		s.fws[databaseConnectionId] = fw
	}

	if _, err = fw.Write(data); err != nil {
		return err
	}

	return nil
}

func (s *QueueStorage) StartConsumer() error {
	s.wg.Add(s.Workers)
	for i := 0; i < s.Workers; i++ {
		log.Info().Int("pid", i).Msg("Starting Consumer")
		go s.consumeMessages(i)
	}

	return nil
}

func (s *QueueStorage) StopConsumer() error {
	log.Info().Msg("Shutting down data importer")
	s.done <- true
	s.wg.Wait()
	return nil
}

func (s *QueueStorage) insertMessage(msg models.FileUploadMessage) (retErr error) {
	// TODO: use correct values
	dbID := "dummy"
	tableName := "my_table"

	conn := s.DB.GetDatabaseConnection(dbID)
	dest := destinations.GetDestination(conn)
	if dest == nil {
		return fmt.Errorf("QueueStorage.insertMessage: Cannot get destination for '%s'", dbID)
	}

	fn := filepath.Join(s.DataDir, filepath.Base(msg.Path))
	// _try_ to ensure the directory exists.
	// it can fail due to permissions, etc. so defer to the create() error
	os.MkdirAll(s.DataDir, 0700)
	file, err := os.Create(fn)
	if err != nil {
		return fmt.Errorf("QueueStorage.insertMessage: Cannot create '%s': %w", fn, err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Error().
				Err(err).
				Str("file", fn).
				Str("db", dbID).
				Str("table", tableName).
				Msg("Closing data file failed")
		}

		// keep the file if insertion failed
		if retErr != nil {
			os.Remove(file.Name())
		}
	}()

	if err := s.storage.Download(msg.Path, file); err != nil {
		return fmt.Errorf("QueueStorage.insertMessage: Cannot download '%s': %w", msg.Path, err)
	}

	if err = dest.InsertBatchFromNDJson(tableName, file); err != nil {
		log.Error().
			Err(err).
			Str("file", fn).
			Str("db", dbID).
			Str("table", tableName).
			Msg("Unable to save data to db")
	}

	return nil
}

func (s *QueueStorage) consumeMessages(pid int) {
	defer s.wg.Done()

	for {
		select {
		case <-s.done:
			return
		default:
		}

		// Ensure we haven't filled up disk
		// TODO: ensure we have enough disk space for: max file upload size, temporary file for insert statement, add'l overhead
		// Could farm this out to AWS batch with a machine sized for the data.
		if util.FreeDiskSpace(s.DataDir) <= uint64(s.opt.MaxFileSize) {
			log.Error().Int("pid", pid).Msg("Disk is full, not consuming any messages")
			time.Sleep(1 * time.Minute)
			continue
		}

		data, err := s.queue.Dequeue()
		if err != nil {
			if !errors.Is(err, queue.ErrEmpyQueue) {
				log.Error().Int("pid", pid).Err(err).Msg("Could not dequeue message")
			}
			select {
			// TODO: implement polling in the queue backends
			case <-time.After(20 * time.Second):
				continue
			case <-s.done:
				return
			}
		}

		msg := models.FileUploadMessage{}
		if err := json.Unmarshal(data, &msg); err != nil {
			log.Error().Int("pid", pid).Err(err).Bytes("message", data).Msg("Could not parse message")
			continue
		}

		if err := s.insertMessage(msg); err != nil {
			log.Error().
				Int("pid", pid).
				Str("path", msg.Path).
				Str("key", msg.Key).
				Err(err).
				Msg("Cannot insert message")
		}
	}
}
