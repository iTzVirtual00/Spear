package client_vfs

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"spear/config"
	"spear/serve"
	"spear/utils"
	"strings"

	http2 "golang.org/x/net/http2"
)

type SpearClient struct {
	spearConfig    *config.SpearConfig
	identity       *config.Identity
	httpClient     *http.Client
	downloadParams *config.DownloadParams
}

type FileDataResponse struct {
	Name string
	Data []byte
}

// FileType represents whether an entry is a file or directory.
type FileType int

const (
	File FileType = iota
	Dir
)

func NewSpearClient(spearConfig *config.SpearConfig, downloadParams *config.DownloadParams, identity *config.Identity) *SpearClient {
	if identity == nil {
		t, ok := spearConfig.Identities["default"]
		if !ok {
			panic("default identity not found")
		}
		identity = &t
	}

	certPEM := []byte(identity.Cert) // adjust field names as needed
	keyPEM := []byte(identity.Key)
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		log.Fatalf("failed to parse httpClient certificate: %v", err)
	}
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			Certificates:       []tls.Certificate{cert},
			MinVersion:         tls.VersionTLS13,
			InsecureSkipVerify: true, // TODO: set to false in production
			VerifyPeerCertificate: func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
				// Custom verification logic if needed
				return nil
			},
		},
		ForceAttemptHTTP2: true,
	}
	_ = http2.ConfigureTransport(transport)
	client := &http.Client{
		Transport: transport,
	}

	return &SpearClient{
		spearConfig:    spearConfig,
		identity:       identity,
		httpClient:     client,
		downloadParams: downloadParams,
	}
}

func (c *SpearClient) newRequest(path string) (*http.Response, error) {
	//req, err := http.NewRequest("GET", "https://"+c.downloadParams.PeerAddress+path, nil)
	req, err := http.NewRequest("GET", "https://"+c.downloadParams.PeerAddress+path, nil)
	if err != nil {
		log.Printf("NewRequest err: %v", err)
		return nil, err
	}

	req.Header.Set("User-Agent", "Spear")
	req.Header.Set("Accept", "application/json")
	resp, err := c.httpClient.Do(req)
	return resp, err
}

func (c *SpearClient) HealthCheck() (bool, error) {
	resp, err := c.newRequest("/spear")
	if err != nil {
		log.Printf("HealthCheck err: %v", err)
		return false, err
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200, nil
}

func parseJson(body io.Reader, v interface{}) error {
	decoder := json.NewDecoder(body)
	return decoder.Decode(v)
}
func fixPath(path string) string {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return path
}

func (c *SpearClient) GetFiles(path string) (*serve.JsonResponse, error) {
	path = fixPath(path)
	resp, err := c.newRequest("/files" + path + "?meta")
	if err != nil {
		log.Printf("GetFiles(%s) err: %v", path, err)
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		log.Printf("GetFiles(%s) failed with status %d: %s", path, resp.StatusCode, string(bodyBytes))
		return nil, fmt.Errorf("failed to get files: %s", string(bodyBytes))
	}
	var jsonResp serve.JsonResponse
	err = parseJson(resp.Body, &jsonResp)
	if err != nil {
		bodyBytes, _ := io.ReadAll(resp.Body) // TODO: won't work here, resp.Body is already consumed
		log.Printf("GetFiles(%s) parseJson err: %v\nbody: %v", path, err, string(bodyBytes))
		return nil, err
	}
	jsonResp.Contents = utils.Filter(jsonResp.Contents, func(f serve.JsonFileResponse) bool {
		return !utils.PathContainsDotDot(f.Name)
	})
	return &jsonResp, nil
}

func (c *SpearClient) DownloadFile(path string) (*FileDataResponse, error) {
	path = fixPath(path)
	resp, err := c.newRequest("/files" + path)
	if err != nil {
		log.Printf("DownloadFile err: %v", err)
		return nil, err
	}
	contentDisposition := resp.Header.Get("Content-Disposition")
	mediatype, params, err := mime.ParseMediaType(contentDisposition)
	if mediatype != "attachment" {
		return nil, fmt.Errorf("unexpected content-disposition: %s", contentDisposition)
	}
	filename := params["filename"]
	if utils.PathContainsDotDot(filename) {
		return nil, fmt.Errorf("filename contains dot-dot: %s", filename)
	}
	if filename == "" {
		return nil, fmt.Errorf("no filename in content-disposition: %s", contentDisposition)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download file: %s", string(data))
	}
	//log.Printf("DownloadFile data: %s", string(data))

	if err != nil {
		log.Printf("DownloadFile read body err: %v", err)
		return nil, err
	}

	return &FileDataResponse{Data: data, Name: filename}, nil // TODO: streaming? there is no support for partial reads
}
