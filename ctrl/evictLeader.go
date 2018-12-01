package main

import (
	"encoding/json"
	"fmt"
	"github.com/unrolled/render"
	"io"
	"net/http"
	"time"

	"github.com/juju/errors"
	"github.com/zhouqiang-cl/hack/types"
	"github.com/zhouqiang-cl/hack/utils"
)

//func init() {
//	rand.Seed(time.Now().UnixNano())
//}

var (
	storesPrefix     = "pd/api/v1/stores"
	storePrefix      = "pd/api/v1/store"
	schedulersPrefix = "pd/api/v1/schedulers"
)

type evictLeaderHandler struct {
	c  *Manager
	rd *render.Render
}

func newEvictLeaderHandler(c *Manager, rd *render.Render) *evictLeaderHandler {
	return &evictLeaderHandler{
		c:  c,
		rd: rd,
	}
}

func (f *evictLeaderHandler) EvictLeader(w http.ResponseWriter, r *http.Request) {

}

type evictLeaderCtl struct {
	url        string
	httpClient *http.Client
}

func newEvictLeaderCtl(url string, timeout time.Duration) *evictLeaderCtl {
	return &evictLeaderCtl{
		url:        url,
		httpClient: &http.Client{Timeout: timeout},
	}
}

func (e *evictLeaderCtl) start(tikvIP string) error {
	return errors.Trace(e.doEvictLeader(tikvIP))
}

func (e *evictLeaderCtl) doEvictLeader(tikvIP string) error {
	storesInfo, err := e.getStores()
	var storeID uint64
	for _, store := range storesInfo.Stores {
		if store.Store.Address == tikvIP {
			storeID = store.Store.Id
		}
	}

	leaderEvictInfo := getLeaderEvictSchedulerInfo(storeID)
	apiURL := fmt.Sprintf("%s/%s", e.url, schedulersPrefix)
	data, err := json.Marshal(leaderEvictInfo)
	if err != nil {
		return err
	}

	_, err = utils.DoPost(apiURL, data)
	if err != nil {
		return err
	}

	for {
		storeInfo, err := e.getStore(storeID)
		if err != nil {
			return err
		}
		if storeInfo.Status.LeaderCount == 0 {
			break
		}
	}

	return nil
}

func (e *evictLeaderCtl) getStores() (*types.StoresInfo, error) {
	apiURL := fmt.Sprintf("%s/%s", e.url, storesPrefix)
	body, err := utils.DoGet(apiURL)
	if err != nil {
		return nil, err
	}

	storesInfo := types.StoresInfo{}
	err = json.Unmarshal(body, &storesInfo)
	if err != nil {
		return nil, err
	}

	return &storesInfo, nil
}

func (e *evictLeaderCtl) getStore(storeID uint64) (*types.StoreInfo, error) {
	apiURL := fmt.Sprintf("%s/%s/%d", e.url, storePrefix, storeID)
	body, err := utils.DoGet(apiURL)
	if err != nil {
		return nil, err
	}

	storeInfo := types.StoreInfo{}
	err = json.Unmarshal(body, &storeInfo)
	if err != nil {
		return nil, err
	}

	return &storeInfo, nil
}

// DeferClose captures the error returned from closing (if an error occurs).
// This is designed to be used in a defer statement.
func DeferClose(c io.Closer, err *error) {
	if cerr := c.Close(); cerr != nil && *err == nil {
		*err = cerr
	}
}

func getLeaderEvictSchedulerInfo(storeID uint64) *types.SchedulerInfo {
	return &types.SchedulerInfo{Name: "evict-leader-scheduler", StoreID: storeID}
}
