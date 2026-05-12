package main

import (
	"context"
	"net"
	"strconv"

	"github.com/segmentio/kafka-go"
)

// PartitionInfo defines partition tracking metrics
type PartitionInfo struct {
	ID           int   `json:"id"`
	StartOffset  int64 `json:"start_offset"`
	EndOffset    int64 `json:"end_offset"`
	MessageCount int64 `json:"message_count"`
}

// EnsureTopicExists verifies if a topic exists on startup. If missing, automatically creates it.
func EnsureTopicExists(broker, topic string, partitions int, logHub *LogHub) error {
	logHub.Log("system", "INFO", "Probing Kafka Broker at %s to verify topic '%s'...", broker, topic)

	conn, err := kafka.Dial("tcp", broker)
	if err != nil {
		logHub.Log("system", "ERROR", "Failed to dial Kafka broker: %v", err)
		return err
	}
	defer conn.Close()

	brokerPartitions, err := conn.ReadPartitions(topic)
	if err == nil && len(brokerPartitions) > 0 {
		logHub.Log("system", "INFO", "Topic '%s' already exists with %d partitions. Skipping creation.", topic, len(brokerPartitions))
		return nil
	}

	controller, err := conn.Controller()
	if err != nil {
		logHub.Log("system", "ERROR", "Failed to retrieve broker controller: %v", err)
		return err
	}

	controllerAddr := net.JoinHostPort(controller.Host, strconv.Itoa(controller.Port))
	logHub.Log("system", "INFO", "Connecting to cluster controller at %s...", controllerAddr)

	controllerConn, err := kafka.Dial("tcp", controllerAddr)
	if err != nil {
		logHub.Log("system", "ERROR", "Failed to dial controller: %v", err)
		return err
	}
	defer controllerConn.Close()

	topicConfigs := []kafka.TopicConfig{
		{
			Topic:             topic,
			NumPartitions:     partitions,
			ReplicationFactor: 1,
		},
	}

	err = controllerConn.CreateTopics(topicConfigs...)
	if err != nil {
		logHub.Log("system", "ERROR", "Failed to create topic: %v", err)
		return err
	}

	logHub.Log("system", "INFO", "Successfully created Kafka topic '%s' with %d partitions.", topic, partitions)
	return nil
}

// GetTopicPartitions queries partitions metadata and gets latest offsets from leader nodes
func GetTopicPartitions(broker, topic string) []PartitionInfo {
	conn, err := kafka.Dial("tcp", broker)
	if err != nil {
		return nil
	}
	defer conn.Close()

	partitions, err := conn.ReadPartitions(topic)
	if err != nil {
		return nil
	}

	infos := make([]PartitionInfo, len(partitions))
	for i, p := range partitions {
		pConn, err := kafka.DialPartition(context.Background(), "tcp", broker, p)
		if err != nil {
			infos[i] = PartitionInfo{ID: p.ID}
			continue
		}

		first, last, err := pConn.ReadOffsets()
		pConn.Close()
		if err != nil {
			infos[i] = PartitionInfo{ID: p.ID}
			continue
		}

		infos[i] = PartitionInfo{
			ID:           p.ID,
			StartOffset:  first,
			EndOffset:    last,
			MessageCount: last - first,
		}
	}
	return infos
}
