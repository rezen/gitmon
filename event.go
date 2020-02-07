package gitmon


import (
	// "fmt"
	// "time"
	// "github.com/jinzhu/gorm"
	// "strings"
	"github.com/asaskevich/EventBus"
)

type BetterBus struct {
	EventBus.Bus
}

// Overwrite the publish method to "fake" wildcard events
/*
func (b *BetterBus) Publish(topic string, args ...interface{}) {
	wildcardArgs := []interface {}{topic}
	for _, arg := range args {
		wildcardArgs = append(wildcardArgs, arg)
	}

	if strings.Contains(topic, ".") {
		wildcardTopicArgs := make([]interface{}, len(wildcardArgs))
		copy(wildcardTopicArgs, wildcardArgs)
		parts := strings.Split(topic, ".")
		topicSuffix := strings.Join(parts[1:len(parts)], ".")
		wildcardTopicArgs[0] = topicSuffix
		fmt.Println("publish-1", wildcardTopicArgs)
		// sb.Bus.Publish(parts[0] + ".*", wildcardTopicArgs...)
		fmt.Println("nah")
	}
	fmt.Println("publish-2")
	// b.Bus.Publish("*", wildcardArgs...)
	fmt.Println("publish-3")
	b.Bus.Publish(topic, args...)
}*/