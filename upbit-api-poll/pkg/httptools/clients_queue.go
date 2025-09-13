package httptools

import (
	"log/slog"
	"time"

	"github.com/Shadow-Web3-development-studio/listings/upbit-api-poll/pkg/ipscrap"
)

type ClientsQueuePool struct {
	members            chan clientsQueueMember
	clientRestInterval time.Duration
}

type clientsQueueMember struct {
	clients        []Client
	lastAcquiredAt time.Time
}

func NewClientsQueuePool(flatClients []Client, clientRestInterval time.Duration) *ClientsQueuePool {
	ipScrapper := ipscrap.New("abd01eca341ef3")
	clientsByLocation := map[string][]Client{}
	for _, client := range flatClients {
		location, err := ipScrapper.GetLocation(client.IPAddress())
		if err != nil {
			location = "unknown"
		}

		clientsByLocation[location] = append(clientsByLocation[location], client)
	}

	q := ClientsQueuePool{
		members:            make(chan clientsQueueMember, len(flatClients)),
		clientRestInterval: clientRestInterval,
	}

	initTime := time.Now()

	for len(clientsByLocation) > 0 {
		clientsGroup := []Client{}
		locationsInGroup := []string{}

		for location, clients := range clientsByLocation {
			if len(clients) == 0 {
				delete(clientsByLocation, location)
				continue
			}

			clientsGroup = append(clientsGroup, clients[0])
			locationsInGroup = append(locationsInGroup, location)

			clientsByLocation[location] = clients[1:]
		}

		if len(clientsGroup) == 0 {
			continue
		}

		q.members <- clientsQueueMember{clients: clientsGroup, lastAcquiredAt: initTime}
		slog.Info("clients group", "count", len(clientsGroup), "locations", locationsInGroup)
	}

	return &q
}

func (q *ClientsQueuePool) Acquire() ([]Client, ReleaseFunc) {
	member := <-q.members
	acquiredAt := time.Now()

	elapsed := time.Since(member.lastAcquiredAt)
	if elapsed < q.clientRestInterval {
		time.Sleep(q.clientRestInterval - elapsed)
	}

	release := func() {
		q.members <- clientsQueueMember{clients: member.clients, lastAcquiredAt: acquiredAt}
	}

	return member.clients, release
}

func (q *ClientsQueuePool) Len() int {
	return len(q.members)
}
