package resources_test

import (
	. "github.com/cloudfoundry/cli/cf/api/resources"

	"github.com/cloudfoundry/cli/cf/models"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Quotas", func() {
	Describe("ToFields", func() {
		var resource QuotaResource

		BeforeEach(func() {
			resource = QuotaResource{
				Resource: Resource{
					Metadata: Metadata{
						GUID: "my-guid",
						URL:  "url.com",
					},
				},
				Entity: models.QuotaResponse{
					GUID:                    "my-guid",
					Name:                    "my-name",
					MemoryLimit:             1024,
					InstanceMemoryLimit:     5,
					RoutesLimit:             10,
					ServicesLimit:           5,
					NonBasicServicesAllowed: true,
					AppInstanceLimit:        "10",
				},
			}
		})

		Describe("ReservedRoutePorts", func() {
			Context("when it is provided by the API", func() {
				BeforeEach(func() {
					resource.Entity.ReservedRoutePorts = "5"
				})

				It("returns back the value", func() {
					fields := resource.ToFields()
					Expect(fields.ReservedRoutePorts).To(Equal(5))
				})
			})

			Context("when it is *not* provided by the API", func() {
				It("should equal 0", func() {
					fields := resource.ToFields()
					Expect(fields.ReservedRoutePorts).To(Equal(0))
				})
			})
		})
	})
})