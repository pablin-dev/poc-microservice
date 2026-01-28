package e2e_test

import (
	"log"
	"os"
	"time"

	"kafka-soap-e2e-test/tests/clients"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Mountebank E2E Tests", Ordered, func() {
	var mountebankClient *clients.MountebankClient
	var mountebankBaseURL string

	BeforeAll(func() {
		defer GinkgoRecover()
		log.SetOutput(GinkgoWriter)

		mountebankBaseURL = os.Getenv("MOUNTEBANK_BASE_URL")
		if mountebankBaseURL == "" {
			mountebankBaseURL = "http://127.0.0.1:2525" // Default for local execution
		}

		mountebankClient = clients.NewMountebankClient(mountebankBaseURL)
		Expect(mountebankClient).NotTo(BeNil(), "Mountebank client should not be nil")

		err := mountebankClient.WaitForMountebank(30 * time.Second) // Give Mountebank up to 30 seconds
		Expect(err).NotTo(HaveOccurred(), "Mountebank did not become ready")

		// Clean up any existing imposters before running tests
		err = mountebankClient.DeleteAllImpostors()
		Expect(err).NotTo(HaveOccurred(), "Failed to delete all imposters before tests")
	})

	AfterAll(func() {
		defer GinkgoRecover()
		// Clean up imposters after all tests are done
		err := mountebankClient.DeleteAllImpostors()
		Expect(err).NotTo(HaveOccurred(), "Failed to delete all imposters after tests")
	})

	Context("Mountebank API connectivity and basic functionality", func() {
		It("should successfully get an empty list of imposters initially", func() {
			impostersResponse, err := mountebankClient.GetAllImpostors()
			Expect(err).NotTo(HaveOccurred(), "Failed to get all imposters")
			Expect(impostersResponse).NotTo(BeNil(), "Imposters response should not be nil")
			Expect(impostersResponse.Imposters).To(BeEmpty(), "Initially, there should be no imposters")
		})
	})
})
