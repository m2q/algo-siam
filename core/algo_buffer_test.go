package core

import (
	"github.com/algorand/go-algorand-sdk/client/v2/common/models"
	"github.com/m2q/aema/core/client"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
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

func TestAlgorandBuffer_DeleteAppsWhenTooMany(t *testing.T) {
	c := client.CreateAlgorandClientMock("", "")
	c.CreateDummyApps(6, 18, 32)
	buffer, _ := CreateAlgorandBuffer(c, client.GeneratePrivateKey64())
	go buffer.Manage()

	acc, _ := c.AccountInformation("", nil)
	iter := 0
	for len(acc.CreatedApps) != 1 || !client.FulfillsSchema(acc.CreatedApps[0]) {
		select{
			case <- time.After(500 * time.Millisecond):
				t.Fatalf("Manage() didn't return to channel in time")
			case <- buffer.AppChannel:
				acc, _ = c.AccountInformation("", nil)
		}
		if iter > 3 {
			t.Fatalf("loop condition not fulfilled after 3 channel writes")
		}
		iter++
	}
}