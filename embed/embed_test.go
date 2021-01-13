// Copyright 2016 The etcd Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package embed

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/url"
	"strings"
	"testing"
)

func TestMultipleEmbededEtcdInOneProcess(t *testing.T) {

	clusterSize := 3
	cfgs := make([]*Config, clusterSize)
	clusterURLs := make([]string, clusterSize)

	for i := range cfgs {

		peer, err := net.Listen("tcp", ":0")
		panicOn(err)
		client, err := net.Listen("tcp", ":0")
		panicOn(err)

		peerPort := peer.Addr().(*net.TCPAddr).Port
		clientPort := client.Addr().(*net.TCPAddr).Port

		fmt.Printf(" for cfg i=%v, peerPort=%v,  clientPort=%v\n", i, peerPort, clientPort)

		// for cfg i=0, peerPort=32845,  clientPort=40167
		// for cfg i=1, peerPort=46249,  clientPort=40635
		// for cfg i=2, peerPort=39757,  clientPort=45535

		scheme := "http"
		peerURL, _ := url.Parse(fmt.Sprintf("%s://localhost:%d", scheme, peerPort))
		clientURL, _ := url.Parse(fmt.Sprintf("%s://localhost:%d", scheme, clientPort))

		dir, err := ioutil.TempDir("", "embed-test-*")
		panicOn(err)

		name := fmt.Sprintf("server%v", i)

		cfg := NewConfig()
		//cfg.Logger = "zap"
		cfg.LogOutputs = []string{"stdout"}
		cfg.Debug = false
		cfg.Name = name
		cfg.Dir = dir
		cfg.LPUrls = []url.URL{*peerURL}
		cfg.LCUrls = []url.URL{*clientURL}

		cfg.APUrls = cfg.LPUrls
		cfg.ACUrls = cfg.LCUrls

		cfg.LPeerSocket = []*net.TCPListener{peer.(*net.TCPListener)}
		cfg.LClientSocket = []*net.TCPListener{client.(*net.TCPListener)}
		//cfg.ClusterName = "bartholemuuuuu"

		clusterURLs[i] = fmt.Sprintf("%s=%s", name, peerURL)
		cfgs[i] = cfg
	}
	for i := range cfgs {
		cfgs[i].InitialCluster = strings.Join(clusterURLs, ",")

		err := cfgs[i].Validate()
		if err != nil {
			t.Errorf("expected nil error, got %v", err)
		}

		etcd, err := StartEtcd(cfgs[i])
		if err != nil {
			panic(err)
		}
		_ = etcd
	}

	select {}
}

/*
func GetAvailPort() int {
	l, _ := net.Listen("tcp", ":0")
	r := l.Addr()
	l.Close()
	return r.(*net.TCPAddr).Port
}
*/

func panicOn(err error) {
	if err != nil {
		panic(err)
	}
}
