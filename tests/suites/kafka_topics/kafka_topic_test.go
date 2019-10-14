package kafka_topics

import (
	"fmt"
	"testing"

	. "github.com/mesosphere/kudo-kafka-operator/tests/suites"

	"github.com/mesosphere/kudo-kafka-operator/tests/utils"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"
)

var topicName string

var _ = Describe("KafkaTest", func() {
	Describe("[Kafka Sanity Checks]", func() {
		Context("Default installation", func() {
			It("statefulset should have 3 replicas with status READY", func() {
				err := utils.KClient.WaitForStatefulSetReadyReplicasCount(DefaultZkStatefulSetName, TestNamespace, 3, 240)
				Expect(err).To(BeNil())
				err = utils.KClient.WaitForStatefulSetReadyReplicasCount(DefaultKafkaStatefulSetName, TestNamespace, 3, 240)
				Expect(err).To(BeNil())
				Expect(utils.KClient.GetStatefulSetCount(DefaultKafkaStatefulSetName, TestNamespace)).To(Equal(3))
			})
			It("write in broker-3 a message with replication 3 and scale down to 3 brokers and read message from another broker", func() {
				kafkaClient := utils.NewKafkaClient(utils.KClient, &utils.KafkaClientConfiguration{
					Namespace: utils.String(TestNamespace),
				})
				topicSuffix, _ := utils.GetRandString(6)
				topicName := fmt.Sprintf("test-topic-%s", topicSuffix)
				err := utils.KClient.UpdateInstancesCount(DefaultKudoKafkaInstance, TestNamespace, 4)
				Expect(err).To(BeNil())
				err = utils.KClient.WaitForStatefulSetReadyReplicasCount(DefaultKafkaStatefulSetName, TestNamespace, 4, 240)
				Expect(err).To(BeNil())
				out, err := kafkaClient.CreateTopic(GetBrokerPodName(3), DefaultContainerName, topicName, "0:1:3")
				Expect(err).To(BeNil())
				Expect(out).To(ContainSubstring("Created topic"))
				messageToTest := "ReplicatedMessage"
				_, err = kafkaClient.WriteInTopic(GetBrokerPodName(3), DefaultContainerName, topicName, messageToTest)
				Expect(err).To(BeNil())
				out, err = kafkaClient.ReadFromTopic(GetBrokerPodName(3), DefaultContainerName, topicName, messageToTest)
				Expect(err).To(BeNil())
				Expect(out).To(ContainSubstring(messageToTest))
				err = utils.KClient.UpdateInstancesCount(DefaultKudoKafkaInstance, TestNamespace, 3)
				Expect(err).To(BeNil())
				err = utils.KClient.WaitForStatefulSetReadyReplicasCount(DefaultKafkaStatefulSetName, TestNamespace, 3, 240)
				// in case the broker with id 3 was the active controller we should wait for the new active controller
				kafkaClient.WaitForBrokersToBeRegisteredWithService(GetBrokerPodName(0), DefaultContainerName, 100)
				kafkaClient.WaitForBrokersToBeRegisteredWithService(GetBrokerPodName(1), DefaultContainerName, 100)
				kafkaClient.WaitForBrokersToBeRegisteredWithService(GetBrokerPodName(2), DefaultContainerName, 100)
				out, err = kafkaClient.ReadFromTopic(GetBrokerPodName(0), DefaultContainerName, topicName, messageToTest)
				Expect(err).To(BeNil())
				Expect(out).To(ContainSubstring(messageToTest))
			})
		})
	})
})

var _ = BeforeSuite(func() {
	utils.TearDown(TestNamespace)
	utils.KClient.CreateNamespace(TestNamespace, false)
	utils.Setup(TestNamespace)
})

var _ = AfterSuite(func() {
	utils.TearDown(TestNamespace)
})

func TestService(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter(fmt.Sprintf("%s-junit.xml", "kafka-topics"))
	RunSpecsWithDefaultAndCustomReporters(t, "KafkaTopics Suite", []Reporter{junitReporter})
}