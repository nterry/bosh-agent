package infrastructure_test

import (
	"errors"
	"encoding/base64"
	
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	fakesys "github.com/cloudfoundry/bosh-agent/system/fakes"
	fakeinf "github.com/cloudfoundry/bosh-agent/infrastructure/fakes"
	boshlog "github.com/cloudfoundry/bosh-agent/logger"

	. "github.com/cloudfoundry/bosh-agent/infrastructure"
)

var _ = Describe("AzureFileMetadataService", func() {
	var (
		fs              *fakesys.FakeFileSystem
		dnsResolver     *fakeinf.FakeDNSResolver
		metadataService MetadataService
	)

	BeforeEach(func() {
		fs = fakesys.NewFakeFileSystem()
		logger := boshlog.NewLogger(boshlog.LevelNone)
		dnsResolver = &fakeinf.FakeDNSResolver{}
		metadataService = NewAzureFileMetadataService(dnsResolver, fs, "fake-userdata-file-path", "fake-goalstate-file-path", "fake-ovfenv-file-path", logger)
	})

	Describe("GetPublicKey", func() {
		Context("when userdata file exists", func() {
			BeforeEach(func() {
				userDataContents := `testdata<UserName>fake-user</UserName>testdata`
				fs.WriteFileString("fake-ovfenv-file-path", userDataContents)
				
				publicKey := "fake-openssh-key"
				fs.WriteFileString("/home/fake-user/.ssh/authorized_keys", publicKey)
			})

			It("returns public key", func() {
				instanceID, err := metadataService.GetPublicKey()
				Expect(err).NotTo(HaveOccurred())
				Expect(instanceID).To(Equal("fake-openssh-key"))
			})
		})

		Context("when userdata file does not exist", func() {
			It("returns an error", func() {
				_, err := metadataService.GetPublicKey()
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("GetInstanceID", func() {
		Context("when goalstate file exists", func() {
			BeforeEach(func() {
				goalstateContents := `     <InstanceId>fake-instance-id</InstanceId>`
				fs.WriteFileString("fake-goalstate-file-path", goalstateContents)
			})

			It("returns instance id", func() {
				instanceID, err := metadataService.GetInstanceID()
				Expect(err).NotTo(HaveOccurred())
				Expect(instanceID).To(Equal("fake-instance-id"))
			})
		})

		Context("when goalstate file does not exist", func() {
			It("returns an error", func() {
				_, err := metadataService.GetInstanceID()
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("GetServerName", func() {
		Context("when the server name is present in the JSON", func() {
			BeforeEach(func() {
				userDataContents := []byte(`{"server":{"name":"fake-server-name"}}`)
				fs.WriteFileString("fake-userdata-file-path", base64.StdEncoding.EncodeToString(userDataContents))
			})

			It("returns the server name", func() {
				name, err := metadataService.GetServerName()
				Expect(err).ToNot(HaveOccurred())
				Expect(name).To(Equal("fake-server-name"))
			})
		})

		Context("when the server name is not present in the JSON", func() {
			It("returns an error", func() {
				name, err := metadataService.GetServerName()
				Expect(err).To(HaveOccurred())
				Expect(name).To(BeEmpty())
			})
		})
	})

	Describe("GetRegistryEndpoint", func() {
		Context("when userdata contains a dns server", func() {
			BeforeEach(func() {
				userDataContents := []byte(`{
					"registry":{"endpoint":"http://fake-registry.com"},
					"dns":{"nameserver":["fake-dns-server-ip"]}
				}`)
				fs.WriteFileString("fake-userdata-file-path", base64.StdEncoding.EncodeToString(userDataContents))
			})

			Context("when registry endpoint is successfully resolved", func() {
				BeforeEach(func() {
					dnsResolver.RegisterRecord(fakeinf.FakeDNSRecord{
						DNSServers: []string{"fake-dns-server-ip"},
						Host:       "http://fake-registry.com",
						IP:         "http://fake-registry-ip",
					})
				})

				It("returns the successfully resolved registry endpoint", func() {
					endpoint, err := metadataService.GetRegistryEndpoint()
					Expect(err).ToNot(HaveOccurred())
					Expect(endpoint).To(Equal("http://fake-registry-ip"))
				})
			})

			Context("when registry endpoint is not successfully resolved", func() {
				BeforeEach(func() {
					dnsResolver.LookupHostErr = errors.New("fake-lookup-host-err")
				})

				It("returns error because it failed to resolve registry endpoint", func() {
					endpoint, err := metadataService.GetRegistryEndpoint()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("fake-lookup-host-err"))
					Expect(endpoint).To(BeEmpty())
				})
			})
		})

		Context("when userdata does not contain dns servers", func() {
			Context("when userdata file exists", func() {
				BeforeEach(func() {
					userDataContents := []byte(`{"registry":{"endpoint":"http://fake-registry.com"}}`)
					fs.WriteFileString("fake-userdata-file-path", base64.StdEncoding.EncodeToString(userDataContents))
				})

				It("returns registry endpoint", func() {
					registryEndpoint, err := metadataService.GetRegistryEndpoint()
					Expect(err).NotTo(HaveOccurred())
					Expect(registryEndpoint).To(Equal("http://fake-registry.com"))
				})
			})

			Context("when userdata file does not exist", func() {
				It("returns an error", func() {
					_, err := metadataService.GetRegistryEndpoint()
					Expect(err).To(HaveOccurred())
				})
			})
		})
	})

	Describe("IsAvailable", func() {
		It("returns true", func() {
			Expect(metadataService.IsAvailable()).To(BeTrue())
		})
	})
})
