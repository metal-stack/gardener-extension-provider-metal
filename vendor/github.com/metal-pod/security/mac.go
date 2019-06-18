package security

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/go-openapi/runtime"
)

// Our constant header names
const (
	tsHeader   = "X-Date"
	saltHeader = "X-Data-Salt"
)

// WrongHMAC is an error which contains the two hmacs which differ. A
// caller can use this values to log the computed value.
type WrongHMAC struct {
	Got  string
	Want string
}

func (w *WrongHMAC) Error() string {
	return "Wrong HMAC found"
}

func newWrongHMAC(got, want string) *WrongHMAC {
	return &WrongHMAC{Got: got, Want: want}
}

var (
	errIllegalAuthFound = fmt.Errorf("illegal auth found")
	errUnknownAuthFound = fmt.Errorf("unknown authtype found")
)

// A HMACAuth is an authenticator which uses a hmac calculation.
type HMACAuth struct {
	key      []byte
	Lifetime time.Duration
	Type     string
	AuthUser User
}

// HMACAuthOption is a option type for HMACAuth
type HMACAuthOption func(*HMACAuth)

// NewHMACAuth returns a new HMACAuth initialized with the given key. A service
// implementation and a client must share the same key and authtype. The authtype
// will be transported as a scheme in the "Authentication" header. The key
// has to be private and will never be transmitted over the wire.
func NewHMACAuth(authtype string, key []byte, opts ...HMACAuthOption) HMACAuth {
	res := HMACAuth{
		key:      key,
		Lifetime: 15 * time.Second,
		Type:     authtype,
		AuthUser: guest,
	}
	for _, o := range opts {
		o(&res)
	}

	return res
}

// WithUser sets the user which is connected to this HMAC auth.
func WithUser(u User) HMACAuthOption {
	return func(h *HMACAuth) {
		h.AuthUser = u
	}
}

// WithLifetime sets the lifetime which is connected to this HMAC auth. If the
// lifetime is zero, there will be no datetime checking. Do not do this in
// productive code (only useful in tests).
func WithLifetime(max time.Duration) HMACAuthOption {
	return func(h *HMACAuth) {
		h.Lifetime = max
	}
}

func (hma *HMACAuth) createMac(vals ...[]byte) string {
	h := hmac.New(sha256.New, hma.key)
	for _, v := range vals {
		h.Write(v)
	}
	sha := hex.EncodeToString(h.Sum(nil))
	return sha
}

// create returns a a formatted timestamp and the generated HMAC.
func (hma *HMACAuth) create(t time.Time, vals ...[]byte) (string, string) {
	ts := t.UTC().Format(time.RFC3339)
	vals = append([][]byte{[]byte(ts)}, vals...)
	return hma.createMac(vals...), ts
}

// AddAuth adds the needed headers to the given request so the given values in the vals-array
// are correctly signed. This function can be used by a client to enhance the request before
// submitting it.
func (hma *HMACAuth) AddAuth(rq *http.Request, t time.Time, body []byte) {
	salt := randByteString(24)
	mac, ts := hma.create(t, []byte(rq.Method), salt)
	rq.Header.Add(tsHeader, ts)
	rq.Header.Add(saltHeader, string(salt))
	rq.Header.Add("Authorization", hma.Type+" "+mac)
}

// AddAuthToClientRequest to support openapi too
func (hma *HMACAuth) AddAuthToClientRequest(rq runtime.ClientRequest, t time.Time) {
	salt := randByteString(24)
	mac, ts := hma.create(t, []byte(rq.GetMethod()), salt)
	rq.SetHeaderParam(tsHeader, ts)
	rq.SetHeaderParam(saltHeader, string(salt))
	rq.SetHeaderParam("Authorization", hma.Type+" "+mac)
}

// User calculates the hmac from header values. The input-values for the calculation
// are: Date-Header, Request-Method, Request-Content.
// If the result does not match the HMAC in the header, this function returns an error. Otherwise
// it returns the user which is connected to this hmac-auth.
func (hma *HMACAuth) User(rq *http.Request) (*User, error) {
	t := rq.Header.Get(tsHeader)
	auth := rq.Header.Get("Authorization")
	if auth == "" {
		return nil, errNoAuthFound
	}
	splitToken := strings.Split(auth, " ")
	if len(splitToken) != 2 {
		return nil, errIllegalAuthFound
	}
	if strings.TrimSpace(splitToken[0]) != hma.Type {
		return nil, errUnknownAuthFound
	}
	hm := strings.TrimSpace(splitToken[1])
	ts, err := time.Parse(time.RFC3339, t)
	if err != nil {
		return nil, fmt.Errorf("unknown timestamp %q in %q header, use RFC3339: %v", t, tsHeader, err)
	}
	if hma.Lifetime > 0 {
		if time.Since(ts) > hma.Lifetime {
			return nil, fmt.Errorf("the timestamp in your header is too old: %q", t)
		}
	}
	vals := hma.getData(rq)
	calc, _ := hma.create(ts, vals...)
	if calc != hm {
		return nil, newWrongHMAC(hm, calc)
	}
	// lets return a copy of our user so the caller cannot change it
	newuser := hma.AuthUser
	return &newuser, nil
}

func (hma *HMACAuth) getData(rq *http.Request) [][]byte {
	return [][]byte{
		[]byte(rq.Method),
		[]byte(rq.Header.Get(saltHeader)),
	}
}

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

var randByteString = func(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return b
}
