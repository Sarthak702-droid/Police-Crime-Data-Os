package services

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"backend/internal/models"
	"backend/internal/repositories"

	"github.com/google/uuid"
)

type S3ObjectStore struct {
	endpoint, accessKey, secretKey, bucket, region string
	http                                           *http.Client
}

func NewS3ObjectStore(endpoint, accessKey, secretKey, bucket, region string) *S3ObjectStore {
	return &S3ObjectStore{endpoint: strings.TrimRight(endpoint, "/"), accessKey: accessKey, secretKey: secretKey, bucket: bucket, region: region, http: &http.Client{Timeout: 60 * time.Second}}
}

func (s *S3ObjectStore) Put(ctx context.Context, key, contentType string, data []byte) error {
	if err := s.ensureBucket(ctx); err != nil {
		return err
	}
	objectPath := "/" + url.PathEscape(s.bucket) + "/" + escapeObjectKey(key)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, s.endpoint+objectPath, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", contentType)
	s.sign(req, data)
	resp, err := s.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("object storage returned HTTP %d", resp.StatusCode)
	}
	return nil
}
func (s *S3ObjectStore) Get(ctx context.Context, key string) ([]byte, string, error) {
	objectPath := "/" + url.PathEscape(s.bucket) + "/" + escapeObjectKey(key)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.endpoint+objectPath, nil)
	if err != nil {
		return nil, "", err
	}
	s.sign(req, nil)
	resp, err := s.http.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("object storage returned HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 50<<20))
	return data, resp.Header.Get("Content-Type"), err
}
func (s *S3ObjectStore) ensureBucket(ctx context.Context) error {
	bucketPath := "/" + url.PathEscape(s.bucket)
	head, err := http.NewRequestWithContext(ctx, http.MethodHead, s.endpoint+bucketPath, nil)
	if err != nil {
		return err
	}
	s.sign(head, nil)
	resp, err := s.http.Do(head)
	if err != nil {
		return err
	}
	_ = resp.Body.Close()
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	if resp.StatusCode != http.StatusNotFound {
		return fmt.Errorf("bucket check returned HTTP %d", resp.StatusCode)
	}
	put, err := http.NewRequestWithContext(ctx, http.MethodPut, s.endpoint+bucketPath, nil)
	if err != nil {
		return err
	}
	s.sign(put, nil)
	created, err := s.http.Do(put)
	if err != nil {
		return err
	}
	defer created.Body.Close()
	if created.StatusCode < 200 || created.StatusCode >= 300 {
		return fmt.Errorf("bucket creation returned HTTP %d", created.StatusCode)
	}
	return nil
}

func (s *S3ObjectStore) sign(req *http.Request, payload []byte) {
	now := time.Now().UTC()
	amzDate := now.Format("20060102T150405Z")
	date := now.Format("20060102")
	payloadHash := sha256Hex(payload)
	req.Header.Set("x-amz-content-sha256", payloadHash)
	req.Header.Set("x-amz-date", amzDate)
	canonicalHeaders := fmt.Sprintf("host:%s\nx-amz-content-sha256:%s\nx-amz-date:%s\n", req.URL.Host, payloadHash, amzDate)
	signedHeaders := "host;x-amz-content-sha256;x-amz-date"
	canonicalRequest := strings.Join([]string{req.Method, req.URL.EscapedPath(), "", canonicalHeaders, signedHeaders, payloadHash}, "\n")
	scope := date + "/" + s.region + "/s3/aws4_request"
	stringToSign := "AWS4-HMAC-SHA256\n" + amzDate + "\n" + scope + "\n" + sha256Hex([]byte(canonicalRequest))
	dateKey := hmacSHA256([]byte("AWS4"+s.secretKey), date)
	regionKey := hmacSHA256(dateKey, s.region)
	serviceKey := hmacSHA256(regionKey, "s3")
	signingKey := hmacSHA256(serviceKey, "aws4_request")
	signature := hex.EncodeToString(hmacSHA256(signingKey, stringToSign))
	req.Header.Set("Authorization", fmt.Sprintf("AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s", s.accessKey, scope, signedHeaders, signature))
}
func hmacSHA256(key []byte, value string) []byte {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(value))
	return mac.Sum(nil)
}
func sha256Hex(value []byte) string { sum := sha256.Sum256(value); return hex.EncodeToString(sum[:]) }
func escapeObjectKey(key string) string {
	parts := strings.Split(strings.TrimLeft(key, "/"), "/")
	for i := range parts {
		parts[i] = url.PathEscape(parts[i])
	}
	return strings.Join(parts, "/")
}

type EvidenceStorageService struct {
	store *S3ObjectStore
	repo  *repositories.DomainRepository
}

func NewEvidenceStorageService(store *S3ObjectStore, repo *repositories.DomainRepository) *EvidenceStorageService {
	return &EvidenceStorageService{store: store, repo: repo}
}
func (s *EvidenceStorageService) Authorize(caseID, unitID int) (bool, error) {
	return s.repo.CaseInUnit(caseID, unitID)
}
func (s *EvidenceStorageService) Custody(caseID, documentID int) ([]models.EvidenceCustodyEvent, error) {
	return s.repo.ListCustodyEvents(caseID, documentID)
}
func (s *EvidenceStorageService) Upload(ctx context.Context, caseID, employeeID, unitID int, filename, documentType, language, piiLevel, contentType string, data []byte) (*models.CaseDocument, error) {
	if len(data) == 0 {
		return nil, errors.New("empty evidence file")
	}
	allowed, err := s.repo.CaseInUnit(caseID, unitID)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, errors.New("case not found")
	}
	safeName := safeFilename(filename)
	key := fmt.Sprintf("cases/%d/%s-%s", caseID, uuid.NewString(), safeName)
	if err := s.store.Put(ctx, key, contentType, data); err != nil {
		return nil, err
	}
	row := &models.CaseDocument{CaseMasterID: caseID, DocumentType: documentType, StorageURI: "s3://" + s.store.bucket + "/" + key, SHA256: sha256Hex(data), LanguageCode: language, PiiLevel: piiLevel, OriginalName: filename, ContentType: contentType, SizeBytes: int64(len(data)), CreatedBy: employeeID, CreatedAt: time.Now().UTC()}
	if err := s.repo.AddDocument(row); err != nil {
		return nil, err
	}
	_ = s.repo.AddCustodyEvent(&models.EvidenceCustodyEvent{DocumentID: row.DocumentID, CaseID: caseID, EventType: "uploaded", ActorID: employeeID, Notes: "Evidence uploaded; SHA-256 " + row.SHA256, CreatedAt: time.Now().UTC()})
	return row, nil
}
func (s *EvidenceStorageService) Download(ctx context.Context, caseID, documentID, employeeID, unitID int) (*models.CaseDocument, []byte, error) {
	allowed, err := s.repo.CaseInUnit(caseID, unitID)
	if err != nil || !allowed {
		return nil, nil, errors.New("case not found")
	}
	row, err := s.repo.GetDocument(caseID, documentID)
	if err != nil || row == nil {
		return nil, nil, errors.New("document not found")
	}
	prefix := "s3://" + s.store.bucket + "/"
	if !strings.HasPrefix(row.StorageURI, prefix) {
		return nil, nil, errors.New("unsupported storage URI")
	}
	data, contentType, err := s.store.Get(ctx, strings.TrimPrefix(row.StorageURI, prefix))
	if err != nil {
		return nil, nil, err
	}
	if row.ContentType == "" {
		row.ContentType = contentType
	}
	_ = s.repo.AddCustodyEvent(&models.EvidenceCustodyEvent{DocumentID: documentID, CaseID: caseID, EventType: "accessed", ActorID: employeeID, Notes: "Evidence content retrieved", CreatedAt: time.Now().UTC()})
	return row, data, nil
}
func (s *EvidenceStorageService) UpdateMetadata(caseID, documentID, employeeID, unitID int, documentType, language, pii, note string) error {
	allowed, err := s.repo.CaseInUnit(caseID, unitID)
	if err != nil || !allowed {
		return errors.New("case not found")
	}
	values := map[string]interface{}{}
	if documentType != "" {
		values["document_type"] = documentType
	}
	if language != "" {
		values["language_code"] = language
	}
	if pii != "" {
		values["pii_level"] = pii
	}
	if len(values) == 0 {
		return errors.New("no metadata supplied")
	}
	if err := s.repo.UpdateDocument(caseID, documentID, values); err != nil {
		return err
	}
	return s.repo.AddCustodyEvent(&models.EvidenceCustodyEvent{DocumentID: documentID, CaseID: caseID, EventType: "classified", ActorID: employeeID, Notes: note, CreatedAt: time.Now().UTC()})
}
func safeFilename(name string) string {
	name = filepath.Base(name)
	name = strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '.' || r == '-' || r == '_' {
			return r
		}
		return '_'
	}, name)
	if name == "" || name == "." {
		return "evidence.bin"
	}
	runes := []rune(name)
	if len(runes) > 120 {
		name = string(runes[len(runes)-120:])
	}
	return name
}
