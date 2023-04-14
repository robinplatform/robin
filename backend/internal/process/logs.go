package process

import (
	"context"
	"os"
	"path"
	"path/filepath"

	"github.com/nxadm/tail"
	"robinplatform.dev/internal/log"
	"robinplatform.dev/internal/pubsub"
)

func (m *ProcessManager) getLogFilePath(id ProcessId) string {
	processLogsPath := filepath.Join(m.processLogsFolderPath, filepath.FromSlash(id.Category), id.Key+".log")
	return processLogsPath
}

func (r *RHandle) GetLogFile(id ProcessId) (LogFileResult, error) {
	proc, found := r.FindById(id)
	if !found {
		return LogFileResult{}, processNotFound(id)
	}

	var info pubsub.TopicInfo
	if proc.logsTopic != nil {
		info = proc.logsTopic.LockWithInfo()
		defer proc.logsTopic.Unlock()
	}

	path := r.m.getLogFilePath(id)
	f, err := os.ReadFile(path)
	if err != nil {
		return LogFileResult{}, err
	}

	res := LogFileResult{
		Text:    string(f),
		Counter: info.Counter,
	}

	return res, nil
}

func (id ProcessId) LogsTopicId() pubsub.TopicId {
	return pubsub.TopicId{
		Category: path.Join("/logs", id.Category),
		Key:      id.Key,
	}
}

func (manager *ProcessManager) logTopicForProcId(id ProcessId) (*pubsub.Topic[string], error) {
	topicId := id.LogsTopicId()

	topic, err := pubsub.CreateTopic[string](manager.registry, topicId)
	if err != nil {
		logger.Err("error creating logging topic for process", log.Ctx{
			"id":  id,
			"err": err.Error(),
		})
		return nil, err
	}

	return topic, nil
}

type topicTailInfo struct {
	processId ProcessId
	logsTopic *pubsub.Topic[string]
	Context   context.Context
}

func (m *ProcessManager) pipeTailIntoTopic(process topicTailInfo) {
	if process.logsTopic == nil || process.logsTopic.IsClosed() {
		return
	}

	defer process.logsTopic.Close()

	config := tail.Config{
		ReOpen: true,
		Follow: true,
		Logger: tail.DiscardingLogger,
	}
	out, err := tail.TailFile(m.getLogFilePath(process.processId), config)
	if err != nil {
		logger.Err("failed to tail file", log.Ctx{
			"err": err.Error(),
		})
		return
	}

	defer out.Cleanup()

	for {
		select {
		case <-process.Context.Done():
			return

		case line, ok := <-out.Lines:
			if !ok {
				return
			}

			if line.Err != nil {
				logger.Err("got error in tail line", log.Ctx{
					"err": line.Err.Error(),
				})
				continue
			}

			process.logsTopic.Publish(line.Text)
		}
	}
}
