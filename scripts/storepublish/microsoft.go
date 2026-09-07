package main

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type storeClient struct {
	rt      runtime
	token   string
	expires time.Time
}

type serviceError struct {
	status    int
	operation string
}

func (e *serviceError) Error() string {
	return fmt.Sprintf("%s returned HTTP %d", e.operation, e.status)
}

func (s *storeClient) accessToken(ctx context.Context) (string, error) {
	if s.token != "" && s.rt.Now().Add(time.Minute).Before(s.expires) {
		return s.token, nil
	}
	tenant, client, secret := s.rt.Env("MSSTORE_TENANT_ID"), s.rt.Env("MSSTORE_CLIENT_ID"), s.rt.Env("MSSTORE_CLIENT_SECRET")
	if tenant == "" || client == "" || secret == "" {
		return "", fmt.Errorf("configure MSSTORE_TENANT_ID, MSSTORE_CLIENT_ID and MSSTORE_CLIENT_SECRET in the microsoft-store environment")
	}
	endpoint := s.rt.TokenURL
	if endpoint == "" {
		if !regexp.MustCompile(`^[A-Za-z0-9.-]+$`).MatchString(tenant) {
			return "", fmt.Errorf("invalid Microsoft tenant identifier")
		}
		endpoint = "https://login.microsoftonline.com/" + tenant + "/oauth2/token"
	}
	data := url.Values{"grant_type": {"client_credentials"}, "client_id": {client}, "client_secret": {secret}, "resource": {"https://manage.devcenter.microsoft.com"}}.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("invalid token endpoint")
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	clientHTTP := *s.rt.HTTP
	clientHTTP.Jar = nil
	clientHTTP.CheckRedirect = func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }
	resp, err := clientHTTP.Do(req)
	if err != nil {
		return "", fmt.Errorf("Microsoft authentication request failed")
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return "", &serviceError{resp.StatusCode, "Microsoft authentication"}
	}
	var token struct {
		Token   string      `json:"access_token"`
		Expires json.Number `json:"expires_in"`
	}
	if err = json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&token); err != nil {
		return "", fmt.Errorf("invalid Microsoft token response")
	}
	seconds, err := strconv.ParseInt(string(token.Expires), 10, 64)
	if err != nil || seconds <= 0 || seconds > 86400 || token.Token == "" {
		return "", fmt.Errorf("invalid Microsoft token lifetime")
	}
	s.token = token.Token
	s.expires = s.rt.Now().Add(time.Duration(seconds) * time.Second)
	return s.token, nil
}
func (s *storeClient) request(ctx context.Context, method, path string, body object) (object, error) {
	if path != "" && path != "/submissions" && !regexp.MustCompile(`^/submissions/[A-Za-z0-9-]+(/status|/commit)?$`).MatchString(path) {
		return nil, fmt.Errorf("invalid Store submission resource")
	}
	var data []byte
	var err error
	if body != nil {
		data, err = json.Marshal(body)
		if err != nil {
			return nil, err
		}
	}
	endpoint := s.rt.StoreURL + "/applications/" + productID + path
	for attempt := range 3 {
		token, err := s.accessToken(ctx)
		if err != nil {
			return nil, err
		}
		req, err := http.NewRequestWithContext(ctx, method, endpoint, bytes.NewReader(data))
		if err != nil {
			return nil, fmt.Errorf("invalid Microsoft API endpoint")
		}
		req.Header.Set("Authorization", "Bearer "+token)
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		client := *s.rt.HTTP
		client.Jar = nil
		client.CheckRedirect = func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }
		resp, callErr := client.Do(req)
		if callErr != nil {
			if method != http.MethodGet || attempt == 2 {
				return nil, fmt.Errorf("Microsoft %s request outcome is unknown; reconcile before retrying", method)
			}
			if err = s.rt.Wait(ctx, time.Second<<attempt); err != nil {
				return nil, err
			}
			continue
		}
		data, readErr := io.ReadAll(io.LimitReader(resp.Body, (8<<20)+1))
		_ = resp.Body.Close()
		if len(data) > 8<<20 {
			return nil, fmt.Errorf("Microsoft response exceeds size limit")
		}
		if resp.StatusCode == http.StatusUnauthorized && attempt < 2 {
			s.token = ""
			continue
		}
		if readErr != nil {
			return nil, fmt.Errorf("Microsoft response was incomplete; reconcile before retrying")
		}
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			if len(bytes.TrimSpace(data)) == 0 {
				return object{}, nil
			}
			o, err := parseObject(data)
			if err != nil {
				return nil, fmt.Errorf("invalid Microsoft API JSON")
			}
			return o, nil
		}
		retryable := resp.StatusCode == 429 || resp.StatusCode >= 500
		if retryable && method == http.MethodGet && attempt < 2 {
			if err = s.rt.Wait(ctx, retryDelay(resp.Header.Get("Retry-After"), s.rt.Now(), attempt)); err != nil {
				return nil, err
			}
			continue
		}
		return nil, &serviceError{resp.StatusCode, "Microsoft " + method}
	}
	return nil, fmt.Errorf("Microsoft request retry limit reached")
}
func retryDelay(header string, now time.Time, attempt int) time.Duration {
	d := time.Second << attempt
	if seconds, err := strconv.Atoi(header); err == nil && seconds >= 0 {
		d = time.Duration(seconds) * time.Second
	} else if until, err := http.ParseTime(header); err == nil {
		d = until.Sub(now)
	}
	if d < 0 {
		d = 0
	}
	return d
}
func (s *storeClient) application(ctx context.Context) (object, error) {
	app, err := s.request(ctx, http.MethodGet, "", nil)
	if err != nil {
		return nil, err
	}
	if stringField(app, "id") != productID {
		return nil, fmt.Errorf("Microsoft returned the wrong Store product")
	}
	return app, nil
}
func submissionRef(app object, key string) string { return stringField(asObject(app[key]), "id") }
func (s *storeClient) submission(ctx context.Context, id string) (object, error) {
	if id == "" || !regexp.MustCompile(`^[A-Za-z0-9-]+$`).MatchString(id) {
		return nil, fmt.Errorf("invalid Store submission identifier")
	}
	return s.request(ctx, http.MethodGet, "/submissions/"+id, nil)
}
func (s *storeClient) published(ctx context.Context, app object) (object, error) {
	return s.submission(ctx, submissionRef(app, "lastPublishedApplicationSubmission"))
}
func (s *storeClient) upload(ctx context.Context, uploadURL, path string) error {
	u, err := url.Parse(uploadURL)
	if err != nil || u.Host == "" || u.User != nil {
		return fmt.Errorf("invalid Store upload endpoint")
	}
	trusted, _ := url.Parse(s.rt.StoreURL)
	if !(u.Scheme == "https" && strings.HasSuffix(strings.ToLower(u.Hostname()), ".blob.core.windows.net")) && !(trusted != nil && u.Scheme == "http" && u.Host == trusted.Host) {
		return fmt.Errorf("Store upload endpoint is not Azure Blob Storage")
	}
	// The API requires an outer ZIP, even when its only file is an MSIX bundle.
	archive, err := os.CreateTemp(s.rt.Scratch, "submission-*.zip")
	if err != nil {
		return err
	}
	defer func() { _ = archive.Close(); _ = os.Remove(archive.Name()) }()
	z := zip.NewWriter(archive)
	w, err := z.CreateHeader(&zip.FileHeader{Name: bundleName, Method: zip.Store})
	if err != nil {
		return err
	}
	bundle, err := os.Open(filepath.Join(path, bundleName))
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(w, bundle)
	closeErr := bundle.Close()
	if copyErr != nil {
		return copyErr
	}
	if closeErr != nil {
		return closeErr
	}
	if err = z.Close(); err != nil {
		return err
	}
	info, err := archive.Stat()
	if err != nil {
		return err
	}
	for attempt := range 3 {
		if _, err = archive.Seek(0, io.SeekStart); err != nil {
			return err
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPut, uploadURL, io.NopCloser(io.LimitReader(archive, info.Size())))
		if err != nil {
			return fmt.Errorf("invalid Store upload endpoint")
		}
		req.ContentLength = info.Size()
		req.Header.Set("x-ms-blob-type", "BlockBlob")
		req.Header.Set("x-ms-version", "2021-12-02")
		req.Header.Set("Content-Type", "application/octet-stream")
		client := *s.rt.HTTP
		client.Jar = nil
		client.CheckRedirect = func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }
		resp, callErr := client.Do(req)
		if callErr != nil {
			if attempt == 2 {
				return fmt.Errorf("bundle upload outcome is unknown")
			}
			if err = s.rt.Wait(ctx, time.Second<<attempt); err != nil {
				return err
			}
			continue
		}
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<20))
		_ = resp.Body.Close()
		if resp.StatusCode == http.StatusCreated {
			return nil
		}
		if (resp.StatusCode == 429 || resp.StatusCode >= 500) && attempt < 2 {
			if err = s.rt.Wait(ctx, retryDelay(resp.Header.Get("Retry-After"), s.rt.Now(), attempt)); err != nil {
				return err
			}
			continue
		}
		return &serviceError{resp.StatusCode, "bundle upload"}
	}
	return fmt.Errorf("bundle upload retry limit reached")
}
