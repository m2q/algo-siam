package core

import (
	"testing"
	"time"

	"github.com/algorand/go-algorand-sdk/client/v2/common/models"
	"github.com/m2q/aema/core/client"
	"github.com/stretchr/testify/assert"
)

// If HealthCheck and token verification works, expect no errors
func TestAlgorandBuffer_HealthAndTokenPass(t *testing.T) {
	c := client.CreateAlgorandClientMock("", "")
	_, err := CreateAlgorandBuffer(c, client.GeneratePrivateKey64())
	if err != nil {
		t.Errorf("failing health check doesn't return error %s", err)
	}
}

// If the HealthCheck is not working, return error upon buffer creation
func TestAlgorandBuffer_NoHealth(t *testing.T) {
	c := client.CreateAlgorandClientMock("", "")
	c.SetError(true, (*client.AlgorandMock).HealthCheck)
	buffer, err := CreateAlgorandBuffer(c, client.GeneratePrivateKey64())
	if err == nil {
		t.Errorf("failing health check doesn't return error %s", err)
	}
	// buffer should still have created account
	assert.NotEqual(t, models.Account{}, buffer.AccountCrypt)
}

// If the Token Verification is not working, return error upon buffer creation
func TestAlgorandBuffer_IncorrectToken(t *testing.T) {
	c := client.CreateAlgorandClientMock("", "")
	c.SetError(true, (*client.AlgorandMock).Status)
	buffer, err := CreateAlgorandBuffer(c, client.GeneratePrivateKey64())
	if err == nil {
		t.Errorf("failing token verification doesn't return error %s", err)
	}
	// buffer should still have created account
	assert.NotEqual(t, models.Account{}, buffer.AccountCrypt)
}

// Without calling buffer's Manage() function, storing on and loading from
// the buffer is invalid and should result in a panic
func TestAlgorandBuffer_RequireManagement(t *testing.T) {
	c := client.CreateAlgorandClientMock("", "")
	buffer, _ := CreateAlgorandBuffer(c, client.GeneratePrivateKey64())

	shouldPanicGet := func() {
		_, _ = buffer.GetBuffer()
	}
	shouldPanicStore := func() {
		buffer.PutElements(make(map[string]string, 3))
	}
	assert.Panics(t, shouldPanicGet)
	assert.Panics(t, shouldPanicStore)
}

// AppChannel should not return anything if the DeleteApplication function
// returns errors.
func TestAlgorandBuffer_DeletionError(t *testing.T) {
	c := client.CreateAlgorandClientMock("", "")
	c.CreateDummyApps(6, 18, 32)
	buffer, _ := CreateAlgorandBuffer(c, client.GeneratePrivateKey64())
	go buffer.Manage()

	c.SetError(true, (*client.AlgorandMock).DeleteApplication)
	select {
	case <-time.After(50 * time.Millisecond):
		break
	case <-buffer.AppChannel:
		t.Fatalf("AppChannel returned information even though DeleteApplication returns error")
	}
}

// BufferMakesTargetValid is a helper method that tests if the buffer brings
// the target account into a valid application state. A valid state means
// that the account has exactly one application, with the correct schema.
// `maxIter` specifies the maximum number of AppChannel callbacks before
// defaulting to a fatal error.
func BufferMakesTargetValid(t *testing.T, buffer *AlgorandBuffer, c client.AlgorandClient, maxIter int) {
	acc, _ := c.AccountInformation("", nil)

	for i := 0; !client.ValidAccount(acc); i++ {
		select {
		case <-time.After(500 * time.Millisecond):
			t.Fatalf("Manage() didn't mutate application in time")
		case <-buffer.AppChannel:
			acc, _ = c.AccountInformation("", nil)
		}

		if i >= maxIter {
			t.Fatalf("loop condition not fulfilled after %d channel writes", maxIter)
		}
	}
}

func TestAlgorandBuffer_DeleteAppsWhenTooMany(t *testing.T) {
	c := client.CreateAlgorandClientMock("", "")
	c.CreateDummyApps(6, 18, 32)
	buffer, _ := CreateAlgorandBuffer(c, client.GeneratePrivateKey64())
	go buffer.Manage()

	BufferMakesTargetValid(t, buffer, c, 3)
}

func TestAlgorandBuffer_DeletePartial(t *testing.T) {
	c := client.CreateAlgorandClientMock("", "")
	c.CreateDummyAppsWithSchema(models.ApplicationStateSchema{}, 6, 18, 32)

	// Set one application to have correct schema
	g, l := client.GenerateSchemasModel()
	c.Account.CreatedApps[0].Params = models.ApplicationParams{GlobalStateSchema: g, LocalStateSchema: l}
	buffer, _ := CreateAlgorandBuffer(c, client.GeneratePrivateKey64())
	go buffer.Manage()

	BufferMakesTargetValid(t, buffer, c, 2)
}

// Given several applications with the right schema, delete the one that has
// been created most recently
func TestAlgorandBuffer_DeleteNewest(t *testing.T) {
	c := client.CreateAlgorandClientMock("", "")
	c.CreateDummyApps(6, 18, 32)

	c.Account.CreatedApps[0].CreatedAtRound = 200
	c.Account.CreatedApps[1].CreatedAtRound = 50
	c.Account.CreatedApps[2].CreatedAtRound = 150

	buffer, _ := CreateAlgorandBuffer(c, client.GeneratePrivateKey64())
	go buffer.Manage()

	BufferMakesTargetValid(t, buffer, c, 2)

	assert.EqualValues(t, 50, c.Account.CreatedApps[0].CreatedAtRound)
}

func TestAlgorandBuffer_Creation(t *testing.T) {
	c := client.CreateAlgorandClientMock("", "")
	buffer, _ := CreateAlgorandBuffer(c, client.GeneratePrivateKey64())

	go buffer.Manage()

	BufferMakesTargetValid(t, buffer, c, 1)
}
