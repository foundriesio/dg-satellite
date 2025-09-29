// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package gateway

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/foundriesio/dg-satellite/context"
	"github.com/foundriesio/dg-satellite/server"
	storage "github.com/foundriesio/dg-satellite/storage/gateway"
)

type testClient struct {
	t   *testing.T
	db  *storage.DbHandle
	fs  *storage.FsHandle
	gw  *storage.Storage
	e   *echo.Echo
	log *slog.Logger

	uuid string
	cert *x509.Certificate
}

func (c testClient) Do(req *http.Request) *httptest.ResponseRecorder {
	req.TLS = &tls.ConnectionState{
		PeerCertificates: []*x509.Certificate{c.cert},
	}
	req = req.WithContext(context.CtxWithLog(req.Context(), c.log))
	rec := httptest.NewRecorder()
	c.e.ServeHTTP(rec, req)
	return rec
}

func (c testClient) GET(resource string, status int, headers ...string) []byte {
	req := httptest.NewRequest(http.MethodGet, resource, nil)
	c.marshalHeaders(headers, req)
	rec := c.Do(req)
	require.Equal(c.t, status, rec.Code)
	return rec.Body.Bytes()
}

func (c testClient) POST(resource string, status int, data any, headers ...string) []byte {
	req := httptest.NewRequest(http.MethodPost, resource, c.marshalBody(data))
	c.marshalHeaders(headers, req)
	rec := c.Do(req)
	require.Equal(c.t, status, rec.Code)
	return rec.Body.Bytes()
}

func (c testClient) PUT(resource string, status int, data any, headers ...string) []byte {
	req := httptest.NewRequest(http.MethodPut, resource, c.marshalBody(data))
	c.marshalHeaders(headers, req)
	rec := c.Do(req)
	require.Equal(c.t, status, rec.Code)
	return rec.Body.Bytes()
}

func (c testClient) marshalHeaders(headers []string, req *http.Request) {
	require.Zero(c.t, len(headers)%2, "Headers must be a sequence of names and values - even number")
	for i := 0; i < len(headers)/2; i++ {
		req.Header.Add(headers[i*2], headers[i*2+1])
	}
}

func (c testClient) marshalBody(data any) io.Reader {
	if s, ok := data.(string); ok {
		return strings.NewReader(s)
	} else if b, ok := data.([]byte); ok {
		return bytes.NewReader(b)
	} else {
		b, err := json.Marshal(data)
		require.Nil(c.t, err)
		return bytes.NewReader(b)
	}
}

func newTestClient(t *testing.T, isProd bool) *testClient {
	tmpDir := t.TempDir()
	fsS, err := storage.NewFs(tmpDir)
	require.Nil(t, err)
	db, err := storage.NewDb(fsS.Config.DbFile())
	require.Nil(t, err)
	gwS, err := storage.NewStorage(db, fsS)
	require.Nil(t, err)

	log, err := context.InitLogger("debug")
	require.Nil(t, err)

	e := server.NewEchoServer()
	RegisterHandlers(e, gwS)

	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.Nil(t, err)

	uuid := rand.Text() // Base32 encoded 128-bit (16-byte, 26 chars) random string
	subj := pkix.Name{CommonName: uuid}
	if isProd {
		bc := pkix.AttributeTypeAndValue{
			Type:  businessCategoryOid,
			Value: businessCategoryProduction,
		}
		subj.Names = append(subj.Names, bc)           // Parsed names
		subj.ExtraNames = append(subj.ExtraNames, bc) // Marshalled names
	}
	cert := x509.Certificate{
		Subject:   subj,
		PublicKey: priv.Public(),
	}
	tc := testClient{
		t:   t,
		gw:  gwS,
		fs:  fsS,
		db:  db,
		e:   e,
		log: log,

		uuid: uuid,
		cert: &cert,
	}
	return &tc
}

func NewTestClient(t *testing.T) *testClient {
	return newTestClient(t, false)
}

func NewProdTestClient(t *testing.T) *testClient {
	return newTestClient(t, true)
}

func TestApiDevice(t *testing.T) {
	lastSeen := time.Now().Add(-1 * time.Second).Unix()
	tc := NewTestClient(t)
	deviceBytes := tc.GET("/device", 200)
	var device storage.Device
	require.Nil(t, json.Unmarshal(deviceBytes, &device))
	assert.Equal(t, tc.cert.Subject.CommonName, device.Uuid)
	assert.Less(t, lastSeen, device.LastSeen)
}

func TestCheckIn(t *testing.T) {
	apps := "a,b,c"
	hash := "abcd"
	tag := "tag"
	target := "target"
	tc := NewTestClient(t)
	deviceBytes := tc.GET(
		"/device", 200, "x-ats-dockerapps", apps, "x-ats-ostreehash", hash, "x-ats-tags", tag, "x-ats-target", target)

	var d *storage.Device
	require.Nil(t, json.Unmarshal(deviceBytes, &d))
	assert.Equal(t, apps, d.Apps)
	assert.Equal(t, hash, d.OstreeHash)
	assert.Equal(t, tag, d.Tag)
	assert.Equal(t, target, d.TargetName)

	d, err := tc.gw.DeviceGet(tc.uuid)
	require.Nil(t, err)
	assert.Equal(t, apps, d.Apps)
	assert.Equal(t, hash, d.OstreeHash)
	assert.Equal(t, tag, d.Tag)
	assert.Equal(t, target, d.TargetName)

	// Check that fields are not erased on a partial update
	tag = "switch"
	apps = "a,b,d"
	_ = tc.GET("/device", 200, "x-ats-dockerapps", apps, "x-ats-tags", tag)

	d, err = tc.gw.DeviceGet(tc.uuid)
	require.Nil(t, err)
	assert.Equal(t, apps, d.Apps)
	assert.Equal(t, hash, d.OstreeHash)
	assert.Equal(t, tag, d.Tag)
	assert.Equal(t, target, d.TargetName)
}

func TestInfo(t *testing.T) {
	akInfo := "[config]\nkey=value"
	hwInfo := `{"key":"value"}`
	hwInfoBad := `{key=value}`
	nwInfo := `{"hostname":"example.org"}`
	nwInfoBad := `{"hostname":123}`
	stInfo := `{"deviceTime":"2025-09-12T10:00:00Z"}`
	stInfo1 := `{"deviceTime":"2025-09-12T10:00:05Z"}`
	stInfoBad := `{"deviceTime":"2025-09-12 10:00:00"}`

	tc := NewTestClient(t)
	_ = tc.PUT("/system_info", 200, hwInfo)
	_ = tc.PUT("/system_info", 400, hwInfoBad)
	_ = tc.PUT("/system_info/config", 200, akInfo)
	_ = tc.PUT("/system_info/network", 200, nwInfo)
	_ = tc.PUT("/system_info/network", 400, nwInfoBad)
	_ = tc.POST("/apps-states", 200, stInfo)
	_ = tc.POST("/apps-states", 200, stInfo1)
	_ = tc.POST("/apps-states", 400, stInfoBad)

	aboveLimit := string(make([]byte, 100*1024+1)) // 100KB of zeroes + one byte
	_ = tc.PUT("/system_info/config", 413, aboveLimit)

	data, err := tc.fs.Devices.ReadFile(tc.uuid, storage.AktomlFile)
	assert.Nil(t, err)
	assert.Equal(t, akInfo, data)
	data, err = tc.fs.Devices.ReadFile(tc.uuid, storage.HwInfoFile)
	assert.Nil(t, err)
	assert.Equal(t, hwInfo, data)
	data, err = tc.fs.Devices.ReadFile(tc.uuid, storage.NetInfoFile)
	assert.Nil(t, err)
	assert.Equal(t, nwInfo, data)

	states, err := tc.fs.Devices.ListFiles(tc.uuid, storage.StatesPrefix, true)
	require.Nil(t, err)
	assert.Equal(t, 2, len(states))
	exp := []string{stInfo, stInfo1}
	for idx, name := range states {
		data, err = tc.fs.Devices.ReadFile(tc.uuid, name)
		assert.Nil(t, err)
		assert.Equal(t, exp[idx], data)
	}

	// apps states rollover
	for i := 0; i < 15; i++ {
		_ = tc.POST("/apps-states", 200, stInfo1)
	}
	states, err = tc.fs.Devices.ListFiles(tc.uuid, storage.StatesPrefix, true)
	require.Nil(t, err)
	assert.Equal(t, 10, len(states))
	for _, name := range states {
		data, err = tc.fs.Devices.ReadFile(tc.uuid, name)
		assert.Nil(t, err)
		assert.Equal(t, stInfo1, data)
	}
}

func TestEvents(t *testing.T) {
	var (
		eventSatus = `{"id":"dead","deviceTime":"2023-12-12T12:00:00Z",` +
			`"event":{"correlationId":"feed","ecu":"","targetName":"metam","version":"42"},` +
			`"eventType":{"id":"satus","version":123}}`
		eventFinis = `{"id":"beaf","deviceTime":"2023-12-12T12:00:42Z",` +
			`"event":{"correlationId":"feed","ecu":"","targetName":"metam","version":"42"},` +
			`"eventType":{"id":"finis","version":123}}`
		eventBadDate = `{"id":"dodo","deviceTime":"omghf",` +
			`"event":{"correlationId":"feed","ecu":"","targetName":"metam","version":"42"},` +
			`"eventType":{"id":"dies","version":123}}`
		eventFixedDate = strings.Replace(eventBadDate, "omghf", time.Now().UTC().Format(time.RFC3339), 1)
		eventBadId     = `{"id":"","deviceTime":"2023-12-12T12:00:55Z",` +
			`"event":{"correlationId":"feed","ecu":"","targetName":"metam","version":"42"},` +
			`"eventType":{"id":"fraus","version":123}}`
		eventBadCorrId = `{"id":"kiwi","deviceTime":"2023-12-12T12:00:55Z",` +
			`"event":{"correlationId":"","ecu":"","targetName":"metam","version":"42"},` +
			`"eventType":{"id":"fraus","version":123}}`

		eventsGood    = fmt.Sprintf(`[%s,%s]`, eventSatus, eventFinis)
		eventsBadData = fmt.Sprintf(`[%s,%s,%s]`, eventBadDate, eventBadId, eventBadCorrId)
		eventsBadJson = "here we go"
	)

	fmt.Println(eventsGood)
	tc := NewTestClient(t)
	_ = tc.POST("/events", 200, eventsGood)
	_ = tc.POST("/events", 200, eventsBadData)
	_ = tc.POST("/events", 400, eventsBadJson)

	eventsFiles, err := tc.fs.Devices.ListFiles(tc.uuid, storage.EventsPrefix, true)
	require.Nil(t, err)
	assert.Equal(t, 1, len(eventsFiles))
	eventsSaved, err := tc.fs.Devices.ReadFile(tc.uuid, eventsFiles[0])
	assert.Nil(t, err)
	assert.Equal(t, fmt.Sprintf("%s\n%s\n%s\n", eventSatus, eventFinis, eventFixedDate), eventsSaved)
}

func TestTufMeta(t *testing.T) {
	tcCi42 := NewTestClient(t)
	tcCi137 := NewTestClient(t)
	tcProd42 := NewProdTestClient(t)
	tcProd137 := NewProdTestClient(t)

	// Positive test bed
	tests := []struct {
		tc     *testClient // CI vs prod
		name   string      // test name and file data
		tag    string      // x-ats-tags header value
		update string      // update name, set for the device
		role   string      // URL's leaf path and file name
	}{
		{tcCi42, "CI test 1.root.json", "test", "42", "1.root.json"},
		{tcCi42, "CI test 3.root.json", "test", "42", "3.root.json"},
		{tcCi42, "CI test timestamp.json", "test", "42", "timestamp.json"},
		{tcCi42, "CI test snapshot.json", "test", "42", "snapshot.json"},
		{tcCi42, "CI test targets.json", "test", "42", "targets.json"},
		{tcCi137, "CI test 137 targets.json", "test", "137", "targets.json"},
		{tcCi137, "CI beta targets.json", "beta", "137", "targets.json"},
		{tcProd42, "Prod beta 1.root.json", "beta", "42", "1.root.json"},
		{tcProd137, "Prod prod 1.root.json", "prod", "137", "1.root.json"},
		{tcProd137, "Prod beta timestamp.json", "beta", "137", "timestamp.json"},
		{tcProd137, "Prod prod snapshot.json", "prod", "137", "snapshot.json"},
		{tcProd42, "Prod frog targets.json", "frog", "42", "targets.json"},
	}

	// Pre-create devices and set their update names before tests
	visited := make(map[*testClient]string, 4)
	for _, ts := range tests {
		if update, ok := visited[ts.tc]; ok {
			// Update must be equal for the same device across test cases.
			require.Equal(t, update, ts.update, ts.name)
		} else {
			visited[ts.tc] = ts.update
			_ = ts.tc.GET("/device", 200) // This creates the device via auto-register
			stmt, err := ts.tc.db.Prepare("TestUpdateUpdate", "UPDATE devices SET update_name=? WHERE uuid=?")
			require.Nil(t, err, ts.name)
			_, err = stmt.Exec(ts.update, ts.tc.cert.Subject.CommonName)
			require.Nil(t, err, ts.name)
		}
	}

	// Pre-create TUF data before tests
	var err error
	for _, ts := range tests {
		switch ts.tc {
		case tcCi42, tcCi137:
			err = ts.tc.fs.Updates.Ci.Tuf.WriteFile(ts.tag, ts.update, ts.role, ts.name)
		case tcProd42, tcProd137:
			err = ts.tc.fs.Updates.Prod.Tuf.WriteFile(ts.tag, ts.update, ts.role, ts.name)
		}
		require.Nil(t, err, ts.name)
	}

	// Finally, run the test
	for _, ts := range tests {
		t.Run(ts.name, func(t *testing.T) {
			ts.tc.t = t // Use sub-test testing handle
			roleBytes := ts.tc.GET("/repo/"+ts.role, 200, "x-ats-tags", ts.tag)
			assert.Equal(t, ts.name, string(roleBytes))
		})
		ts.tc.t = t // Restore parent test testing handle
	}

	// A few negative tests
	t.Run("Missing five.root.json", func(t *testing.T) {
		tcCi42.t = t
		_ = tcCi42.GET("/repo/five.root.json", 404, "x-ats-tags", "test")
	})
	t.Run("Missing 5.root.json", func(t *testing.T) {
		tcCi42.t = t
		_ = tcCi42.GET("/repo/5.root.json", 404, "x-ats-tags", "test")
	})
	t.Run("Missing targets.json for non-existing tag", func(t *testing.T) {
		tcCi42.t = t
		_ = tcCi42.GET("/repo/targets.json", 404, "x-ats-tags", "zero")
	})
	t.Run("Missing targets.json for existing tag but not matching update", func(t *testing.T) {
		tcCi42.t = t
		_ = tcCi42.GET("/repo/targets.json", 404, "x-ats-tags", "beta")
	})
}
