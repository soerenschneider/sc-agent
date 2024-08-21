package wol

import (
	"errors"
	"fmt"
	"net"

	"github.com/soerenschneider/sc-agent/internal/config"
	"github.com/soerenschneider/sc-agent/internal/metrics"
)

var (
	ErrNoSuchAlias = errors.New("no such alias")
)

type Wol struct {
	aliases       map[string]string
	broadcastAddr string
}

func New(conf config.Wol) (*Wol, error) {
	ret := &Wol{
		aliases:       conf.Aliases,
		broadcastAddr: conf.BroadcastAddr,
	}

	return ret, nil
}

func (w *Wol) WakeUp(alias string) error {
	mac, ok := w.aliases[alias]
	if !ok {
		metrics.WolNoSuchAlias.WithLabelValues(alias).Inc()
		return ErrNoSuchAlias
	}

	return sendWolPacket(mac, w.broadcastAddr)
}

// sendWolPacket sends a Wake-on-LAN packet to the specified MAC address
func sendWolPacket(macAddr string, broadcastAddr string) error {
	mac, err := net.ParseMAC(macAddr)
	if err != nil {
		return fmt.Errorf("invalid MAC address: %v", err)
	}

	packet, err := craftMagicPacket(mac)
	if err != nil {
		return err
	}

	conn, err := net.Dial("udp", broadcastAddr)
	if err != nil {
		return fmt.Errorf("failed to dial UDP: %v", err)
	}
	defer conn.Close()

	_, err = conn.Write(packet)
	if err != nil {
		return fmt.Errorf("failed to send packet: %v", err)
	}

	return nil
}

// craftMagicPacket creates a magic packet for the given MAC address
func craftMagicPacket(mac net.HardwareAddr) ([]byte, error) {
	if len(mac) != 6 {
		return nil, fmt.Errorf("invalid MAC address: %s", mac)
	}

	packet := make([]byte, 102)

	// Start with 6 bytes of 0xFF
	for i := 0; i < 6; i++ {
		packet[i] = 0xFF
	}

	// Followed by 16 repetitions of the MAC address
	for i := 1; i <= 16; i++ {
		copy(packet[i*6:], mac)
	}

	return packet, nil
}
