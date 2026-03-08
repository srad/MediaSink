package requests

import "github.com/srad/mediasink/internal/db"

type ChannelTagsUpdateRequest struct {
	Tags *db.Tags `json:"tags" extensions:"!x-nullable"`
}
