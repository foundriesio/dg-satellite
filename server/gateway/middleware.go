// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// All rights reserved.
// Confidential and Proprietary - Qualcomm Technologies, Inc.
package gateway

import (
	"context"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/pem"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"

	dgctx "github.com/foundriesio/dg-satellite/context"
	storage "github.com/foundriesio/dg-satellite/storage/gateway"
)

type _DeviceKey int

const DeviceKey = _DeviceKey(1)

var (
	businessCategoryOid        = asn1.ObjectIdentifier{2, 5, 4, 15}
	businessCategoryProduction = "production"
)

func (h handlers) authDevice(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		req := c.Request()
		ctx := req.Context()
		log := dgctx.CtxGetLog(ctx)
		tls := c.Request().TLS
		cert := tls.PeerCertificates[0]
		uuid := cert.Subject.CommonName

		isProd := getBusinessCategory(cert.Subject) == businessCategoryProduction
		pub, err := pubkey(cert)
		if err != nil {
			return c.String(http.StatusForbidden, fmt.Sprintf("unable to extract device's public key: %s", err))
		}

		device, err := h.storage.DeviceGet(uuid)

		if err != nil {
			log.Error("Unable to query for device", "uuid", uuid, "error", err)
			return c.String(http.StatusBadGateway, err.Error())
		} else if device == nil {
			device, err = h.storage.DeviceCreate(cert.Subject.CommonName, pub, isProd)
			if err != nil {
				log.Error("Unable to create device", "cn", cert.Subject.CommonName, "error", err)
				return c.String(http.StatusBadGateway, err.Error())
			}
			log.Info("Created device", "uuid", device.Uuid)
		} else if device.Deleted {
			return c.String(http.StatusForbidden, fmt.Sprintf("Device(%s) has been deleted", cert.Subject.CommonName))
		} else if pub != device.PubKey {
			/*if err := device.RotatePubKey(pub); err != nil {
				return c.String(http.StatusForbidden, err.Error())
			}*/
			panic("TODO ROTATE KEY")
		}

		ctx = context.WithValue(c.Request().Context(), DeviceKey, device)
		log = log.With("device", uuid)
		ctx = dgctx.CtxWithLog(ctx, log)
		c.SetRequest(req.WithContext(ctx))

		return next(c)
	}
}

func pubkey(cert *x509.Certificate) (string, error) {
	derBytes, err := x509.MarshalPKIXPublicKey(cert.PublicKey)
	if err != nil {
		return "", err
	}
	block := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: derBytes,
	}
	return string(pem.EncodeToMemory(block)), nil
}

// Golang crypto/x509/pkix package doesn't parse a dozen of standard attributes
func getBusinessCategory(subject pkix.Name) string {
	for _, atv := range subject.Names {
		if businessCategoryOid.Equal(atv.Type) {
			return atv.Value.(string)
		}
	}
	return ""
}

func getDevice(c echo.Context) *storage.Device {
	return c.Request().Context().Value(DeviceKey).(*storage.Device)
}
