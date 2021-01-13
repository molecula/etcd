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
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"go.etcd.io/etcd/clientv3"
	"google.golang.org/grpc/grpclog"
)

func TestMultipleEmbededEtcdInOneProcess(t *testing.T) {

	clusterSize := 3
	cfgs := make([]*Config, clusterSize)
	clusterURLs := make([]string, clusterSize)
	var endpointsP []string
	var endpointsC []string

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

		endpointsC = append(endpointsC, clientURL.String())
		endpointsP = append(endpointsP, peerURL.String())
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
		defer etcd.Close()
	}

	// check the cluster using the client
	clientv3.SetLogger(grpclog.NewLoggerV2(os.Stderr, os.Stderr, os.Stderr))

	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   endpointsC,
		DialTimeout: time.Second * 10,
	})
	panicOn(err)
	defer cli.Close() // make sure to close the client

	_, err = cli.Put(context.TODO(), "foo", "bar")
	panicOn(err)

	resp, err := cli.Get(context.TODO(), "foo")
	panicOn(err)
	obs := string(resp.Kvs[0].Value)
	if obs != "bar" {
		panic(fmt.Sprintf("expected '%v', observerd '%v'", "bar", obs))
	}

	// verify our ports are now bound, both Peer and Client.
	for _, hp := range append(endpointsP, endpointsC...) {
		c, err := net.Dial("tcp", strings.TrimPrefix(hp, "http://"))
		panicOn(err)
		panicOn(c.Close())
	}

	fmt.Printf("DONE! resp='%v'\n", obs)
}

func panicOn(err error) {
	if err != nil {
		panic(err)
	}
}
