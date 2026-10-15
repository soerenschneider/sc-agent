package ports

type Nftables interface {
	AddSetElement(req NftablesAddSetElementRequest) error
}

// NftablesAddSetElementRequest is shared by all adapters, it is validated by the service.
type NftablesAddSetElementRequest struct {
	// Set is the configured alias of the set
	Set     string `json:"set"`
	Element string `json:"element"`
	// Timeout in Go duration format, e.g. "1h30m"
	Timeout string `json:"timeout"`
}
