package kafka_kerberos

import (
	"fmt"
	"testing"

	. "github.com/mesosphere/kudo-kafka-operator/tests/suites"

	"github.com/mesosphere/kudo-kafka-operator/tests/utils"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"
)

var customNamespace = "kerberos-ns"
var krb5Client = &utils.KDCClient{}

var _ = Describe("KafkaTest", func() {
	Describe("[Kafka Kerberos Checks]", func() {
		Context("kerberos-ns installation", func() {
			It("kdc service should have count 1", func() {
				krb5Client.CreateKeytabSecret(utils.GetKafkaKeyabs(customNamespace), "kafka", "base64-kafka-keytab-secret")
				Expect(utils.KClient.CheckIfPodExists("kdc", customNamespace)).To(Equal(true))
				Expect(utils.KClient.GetServicesCount("kdc-service", customNamespace)).To(Equal(1))
			})
			It("Kafka and Zookeeper statefulset should have 3 replicas with status READY", func() {
				err := utils.KClient.WaitForStatefulSetReadyReplicasCount(DefaultZkStatefulSetName, customNamespace, 3, 240)
				Expect(err).To(BeNil())
				err = utils.KClient.WaitForStatefulSetReadyReplicasCount(DefaultKafkaStatefulSetName, customNamespace, 3, 240)
				Expect(err).To(BeNil())
				Expect(utils.KClient.GetStatefulSetCount(DefaultKafkaStatefulSetName, customNamespace)).To(Equal(3))
			})
			It("write and read a message with replication 3 in broker-0", func() {
				kafkaClient := utils.NewKafkaClient(utils.KClient, &utils.KafkaClientConfiguration{
					Namespace:       utils.String(customNamespace),
					KerberosEnabled: true,
				})
				topicSuffix, _ := utils.GetRandString(6)
				topicName := fmt.Sprintf("test-topic-%s", topicSuffix)
				out, err := kafkaClient.CreateTopic(GetBrokerPodName(0), DefaultContainerName, topicName, "0:1:2")
				Expect(err).To(BeNil())
				Expect(out).To(ContainSubstring("Created topic"))
				messageToTest := "KerberosMessage"
				_, err = kafkaClient.WriteInTopic(GetBrokerPodName(0), DefaultContainerName, topicName, messageToTest)
				Expect(err).To(BeNil())
				out, err = kafkaClient.ReadFromTopic(GetBrokerPodName(0), DefaultContainerName, topicName, messageToTest)
				Expect(err).To(BeNil())
				Expect(out).To(ContainSubstring(messageToTest))
			})
		})
	})
})

var _ = BeforeSuite(func() {
	utils.TearDown(customNamespace)
	utils.KClient.CreateNamespace(customNamespace, false)
	krb5Client.SetNamespace(customNamespace)
	krb5Client.Deploy()
	utils.SetupWithKerberos(customNamespace)
})

var _ = AfterSuite(func() {
	utils.TearDown(customNamespace)
	utils.KClient.DeleteNamespace(customNamespace)
})

func TestService(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter(fmt.Sprintf("%s-junit.xml", "kafka-kerberos"))
	RunSpecsWithDefaultAndCustomReporters(t, "KafkaKerberos Suite", []Reporter{junitReporter})
}
