package postgres_test

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	. "github.com/phil-inc/pcommon/pkg/postgres"
	. "github.com/phil-inc/pcommon/pkg/postgres/mocks"
)

var _ = Describe("PgxConnPool", func() {
	It("should successfully create a pool and pull it from a context", func() {
		ctrl := gomock.NewController(GinkgoT())
		ctx := context.Background()

		mockPostgres := NewMockPgxConnPool(ctrl)
		Expect(mockPostgres).NotTo(BeNil())

		ctx = SetPgxConnPoolOnContext(ctx, mockPostgres)

		existingPostgres := GetPgxConnPoolFromContext(ctx)
		Expect(existingPostgres).NotTo(BeNil())
		Expect(existingPostgres).To(BeEquivalentTo(mockPostgres))
	})
})

func TestPostgres(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Postgres Suite")
}
