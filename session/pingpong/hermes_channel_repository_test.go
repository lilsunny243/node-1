/*
 * Copyright (C) 2020 The "MysteriumNetwork/node" Authors.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */
package pingpong

import (
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/mocks"
	"github.com/mysteriumnetwork/node/session/pingpong/event"
	"github.com/mysteriumnetwork/payments/client"
	"github.com/mysteriumnetwork/payments/crypto"
	"github.com/stretchr/testify/assert"
)

func TestHermesChannelRepository_Fetch_returns_errors(t *testing.T) {
	// given
	id := identity.FromAddress("0x0000000000000000000000000000000000000001")
	hermesID = common.HexToAddress("0x00000000000000000000000000000000000000002")
	promiseProvider := &mockHermesPromiseStorage{}
	channelStatusProvider := &mockProviderChannelStatusProvider{}
	repo := NewHermesChannelRepository(promiseProvider, channelStatusProvider, mocks.NewEventBus())

	// when
	channelStatusProvider.channelReturnError = errMock
	promiseProvider.errToReturn = nil
	_, err := repo.Fetch(id, hermesID)
	// then
	assert.Errorf(t, err, "could not get provider channel for %v, hermes %v: %v", mockID, common.Address{}.Hex(), errMock.Error())

	// when
	channelStatusProvider.channelReturnError = nil
	promiseProvider.errToReturn = errMock
	_, err = repo.Fetch(mockID, hermesID)
	// then
	assert.Errorf(t, err, "could not get hermes promise for provider %v, hermes %v: %v", mockID, common.Address{}.Hex(), errMock.Error())

}

func TestHermesChannelRepository_Fetch_handles_no_promise(t *testing.T) {
	// given
	id := identity.FromAddress("0x0000000000000000000000000000000000000001")
	hermesID = common.HexToAddress("0x00000000000000000000000000000000000000002")

	expectedPromise := HermesPromise{}
	promiseProvider := &mockHermesPromiseStorage{
		toReturn:    expectedPromise,
		errToReturn: ErrNotFound,
	}

	expectedChannelStatus := client.ProviderChannel{
		Balance: big.NewInt(1000000000000),
		Settled: big.NewInt(9000000),
		Stake:   big.NewInt(12312323),
	}
	channelStatusProvider := &mockProviderChannelStatusProvider{
		channelToReturn: expectedChannelStatus,
	}

	// when
	repo := NewHermesChannelRepository(promiseProvider, channelStatusProvider, mocks.NewEventBus())
	channel, err := repo.Fetch(id, hermesID)
	assert.NoError(t, err)

	// then
	expectedBalance := new(big.Int).Add(expectedChannelStatus.Balance, expectedChannelStatus.Settled)
	assert.Equal(t, expectedBalance, channel.balance())
	assert.Equal(t, expectedBalance, channel.availableBalance())
}

func TestHermesChannelRepository_Fetch_takes_promise_into_account(t *testing.T) {
	// given
	id := identity.FromAddress("0x0000000000000000000000000000000000000001")
	hermesID = common.HexToAddress("0x00000000000000000000000000000000000000002")

	expectedPromise := HermesPromise{
		Promise: crypto.Promise{Amount: big.NewInt(7000000)},
	}
	promiseProvider := &mockHermesPromiseStorage{
		toReturn: expectedPromise,
	}

	expectedChannelStatus := client.ProviderChannel{
		Balance: big.NewInt(1000000000000),
		Settled: big.NewInt(9000000),
		Stake:   big.NewInt(12312323),
	}
	channelStatusProvider := &mockProviderChannelStatusProvider{
		channelToReturn: expectedChannelStatus,
	}

	// when
	repo := NewHermesChannelRepository(promiseProvider, channelStatusProvider, mocks.NewEventBus())
	channel, err := repo.Fetch(id, hermesID)
	assert.NoError(t, err)

	// then
	added := new(big.Int).Add(expectedChannelStatus.Balance, expectedChannelStatus.Settled)
	expectedBalance := added.Sub(added, expectedPromise.Promise.Amount)
	assert.Equal(t, expectedBalance, channel.balance())
	assert.Equal(t, new(big.Int).Add(expectedChannelStatus.Balance, expectedChannelStatus.Settled), channel.availableBalance())
}

func TestHermesChannelRepository_Fetch_publishesEarningChanges(t *testing.T) {
	// given
	id := identity.FromAddress("0x0000000000000000000000000000000000000001")
	hermesID = common.HexToAddress("0x00000000000000000000000000000000000000002")
	expectedPromise1 := HermesPromise{
		Promise: crypto.Promise{Amount: big.NewInt(7000000)},
	}
	expectedPromise2 := HermesPromise{
		Promise: crypto.Promise{Amount: big.NewInt(8000000)},
	}
	expectedChannelStatus1 := client.ProviderChannel{
		Balance: big.NewInt(1000000000000),
		Settled: big.NewInt(9000000),
		Stake:   big.NewInt(12312323),
	}
	expectedChannelStatus2 := client.ProviderChannel{
		Balance: big.NewInt(1000000000001),
		Settled: big.NewInt(9000001),
		Stake:   big.NewInt(12312324),
	}

	promiseProvider := &mockHermesPromiseStorage{}
	channelStatusProvider := &mockProviderChannelStatusProvider{}
	publisher := mocks.NewEventBus()
	repo := NewHermesChannelRepository(promiseProvider, channelStatusProvider, publisher)

	// when
	promiseProvider.toReturn = expectedPromise1
	channelStatusProvider.channelToReturn = expectedChannelStatus1
	channel, err := repo.Fetch(id, hermesID)
	assert.NoError(t, err)

	// then
	expectedChannel1 := NewHermesChannel(id, hermesID, expectedChannelStatus1, expectedPromise1)
	assert.Equal(t, expectedChannel1, channel)
	assert.Eventually(t, func() bool {
		lastEvent, ok := publisher.Pop().(event.AppEventEarningsChanged)
		if !ok {
			return false
		}
		assert.Equal(
			t,
			event.AppEventEarningsChanged{
				Identity: id,
				Previous: event.Earnings{
					LifetimeBalance:  big.NewInt(0),
					UnsettledBalance: big.NewInt(0),
				},
				Current: event.Earnings{
					LifetimeBalance:  expectedChannel1.lifetimeBalance(),
					UnsettledBalance: expectedChannel1.unsettledBalance(),
				},
			},
			lastEvent,
		)
		return true
	}, 2*time.Second, 10*time.Millisecond)

	// when
	promiseProvider.toReturn = expectedPromise2
	channelStatusProvider.channelToReturn = expectedChannelStatus2
	channel, err = repo.Fetch(id, hermesID)
	assert.NoError(t, err)

	// then
	expectedChannel2 := NewHermesChannel(id, hermesID, expectedChannelStatus2, expectedPromise2)
	assert.Equal(t, expectedChannel2, channel)
	assert.Eventually(t, func() bool {
		lastEvent, ok := publisher.Pop().(event.AppEventEarningsChanged)
		if !ok {
			return false
		}
		assert.Equal(
			t,
			event.AppEventEarningsChanged{
				Identity: id,
				Previous: event.Earnings{
					LifetimeBalance:  expectedChannel1.lifetimeBalance(),
					UnsettledBalance: expectedChannel1.unsettledBalance(),
				},
				Current: event.Earnings{
					LifetimeBalance:  expectedChannel2.lifetimeBalance(),
					UnsettledBalance: expectedChannel2.unsettledBalance(),
				},
			},
			lastEvent,
		)
		return true
	}, 2*time.Second, 10*time.Millisecond)
}
