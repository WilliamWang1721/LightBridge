package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	infraerrors "github.com/WilliamWang1721/LightBridge/internal/pkg/errors"
	"github.com/WilliamWang1721/LightBridge/internal/pkg/pagination"
)

const (
	DistributionKindText          = "text"
	DistributionKindMessage       = "message"
	DistributionKindFile          = "file"
	DistributionKindAccountExport = "account_export"

	MaxDistributionAttachmentBytes = 10 << 20
	MaxDistributionRecipients      = 50000
	MaxDistributionBatchItems      = 100
)

var (
	ErrDistributionNotFound = infraerrors.NotFound("DISTRIBUTION_NOT_FOUND", "distribution not found")
	ErrDistributionInvalid  = infraerrors.BadRequest("DISTRIBUTION_INVALID", "distribution input is invalid")
	ErrDistributionAudience = infraerrors.BadRequest("DISTRIBUTION_AUDIENCE_EMPTY", "distribution audience is empty")
	ErrDistributionTooLarge = infraerrors.BadRequest("DISTRIBUTION_ATTACHMENT_TOO_LARGE", "distribution attachment exceeds the size limit")
	ErrDistributionTooMany  = infraerrors.BadRequest("DISTRIBUTION_TOO_MANY_RECIPIENTS", "distribution recipient count exceeds the limit")
)

type Distribution struct {
	ID              int64          `json:"id"`
	Title           string         `json:"title"`
	Kind            string         `json:"kind"`
	Content         string         `json:"content"`
	FileName        string         `json:"file_name,omitempty"`
	ContentType     string         `json:"content_type,omitempty"`
	FileSize        int64          `json:"file_size"`
	HasAttachment   bool           `json:"has_attachment"`
	Metadata        map[string]any `json:"metadata,omitempty"`
	Audience        map[string]any `json:"audience,omitempty"`
	CreatedBy       *int64         `json:"created_by,omitempty"`
	CreatedAt       time.Time      `json:"created_at"`
	RecipientCount  int64          `json:"recipient_count"`
	ReadCount       int64          `json:"read_count"`
	DownloadCount   int64          `json:"download_count"`
	ReadAt          *time.Time     `json:"read_at,omitempty"`
	DownloadedAt    *time.Time     `json:"downloaded_at,omitempty"`
	RecipientTitle  string         `json:"recipient_title,omitempty"`
	RecipientContent string        `json:"recipient_content,omitempty"`
}

type DistributionRecipient struct {
	UserID          int64
	TitleOverride   string
	ContentOverride string
}

type DistributionAttachment struct {
	FileName    string
	ContentType string
	FileSize    int64
	Ciphertext  []byte
}

type DistributionUserFilters struct {
	Status     string           `json:"status,omitempty"`
	Role       string           `json:"role,omitempty"`
	Search     string           `json:"search,omitempty"`
	GroupName  string           `json:"group_name,omitempty"`
	Activity   string           `json:"activity,omitempty"`
	Attributes map[int64]string `json:"attributes,omitempty"`
}

type DistributionAudienceInput struct {
	All     bool                     `json:"all,omitempty"`
	UserIDs []int64                  `json:"user_ids,omitempty"`
	Emails  []string                 `json:"emails,omitempty"`
	Filters *DistributionUserFilters `json:"filters,omitempty"`
	Lines   string                   `json:"lines,omitempty"`
}

type CreateDistributionInput struct {
	Title       string                    `json:"title"`
	Kind        string                    `json:"kind"`
	Content     string                    `json:"content"`
	FileName    string                    `json:"file_name,omitempty"`
	ContentType string                    `json:"content_type,omitempty"`
	FileData    []byte                    `json:"-"`
	FileBase64  string                    `json:"file_base64,omitempty"`
	Metadata    map[string]any            `json:"metadata,omitempty"`
	Audience    DistributionAudienceInput `json:"audience"`
	ActorID     *int64                    `json:"-"`
}

type DistributionAudienceUser struct {
	ID       int64  `json:"id"`
	Email    string `json:"email"`
	Username string `json:"username"`
}

type DistributionBatchItemResult struct {
	Index          int    `json:"index"`
	DistributionID int64  `json:"distribution_id,omitempty"`
	RecipientCount int64  `json:"recipient_count,omitempty"`
	Error          string `json:"error,omitempty"`
}

type DistributionBatchResult struct {
	Succeeded int                           `json:"succeeded"`
	Failed    int                           `json:"failed"`
	Items     []DistributionBatchItemResult `json:"items"`
}

type DistributionRepository interface {
	Create(ctx context.Context, distribution *Distribution, attachment *DistributionAttachment, recipients []DistributionRecipient) error
	ListAdmin(ctx context.Context, params pagination.PaginationParams) ([]Distribution, *pagination.PaginationResult, error)
	ListForUser(ctx context.Context, userID int64, params pagination.PaginationParams, unreadOnly bool) ([]Distribution, *pagination.PaginationResult, error)
	GetByID(ctx context.Context, id int64) (*Distribution, error)
	GetForUser(ctx context.Context, id, userID int64) (*Distribution, error)
	GetAttachment(ctx context.Context, id int64) (*DistributionAttachment, error)
	GetAttachmentForUser(ctx context.Context, id, userID int64) (*DistributionAttachment, error)
	MarkRead(ctx context.Context, id, userID int64, at time.Time) error
	MarkDownloaded(ctx context.Context, id, userID int64, at time.Time) error
	Delete(ctx context.Context, id int64) error
}

type DistributionService struct {
	repo      DistributionRepository
	userRepo  UserRepository
	encryptor SecretEncryptor
}

func NewDistributionService(repo DistributionRepository, userRepo UserRepository, encryptor SecretEncryptor) *DistributionService {
	return &DistributionService{repo: repo, userRepo: userRepo, encryptor: encryptor}
}

func (s *DistributionService) Create(ctx context.Context, input CreateDistributionInput) (*Distribution, error) {
	input.Title = strings.TrimSpace(input.Title)
	input.Kind = strings.ToLower(strings.TrimSpace(input.Kind))
	input.Content = strings.TrimSpace(input.Content)
	input.FileName = sanitizeDistributionFileName(input.FileName)
	input.ContentType = strings.TrimSpace(input.ContentType)

	if input.Title == "" || len([]rune(input.Title)) > 200 || !isDistributionKind(input.Kind) {
		return nil, ErrDistributionInvalid
	}
	if len(input.FileData) == 0 && strings.TrimSpace(input.FileBase64) != "" {
		decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(input.FileBase64))
		if err != nil {
			return nil, ErrDistributionInvalid.WithCause(err)
		}
		input.FileData = decoded
	}
	if len(input.FileData) > MaxDistributionAttachmentBytes {
		return nil, ErrDistributionTooLarge
	}
	if input.Kind == DistributionKindText || input.Kind == DistributionKindMessage {
		if input.Content == "" {
			return nil, ErrDistributionInvalid
		}
	}
	if input.Kind == DistributionKindFile || input.Kind == DistributionKindAccountExport {
		if len(input.FileData) == 0 || input.FileName == "" {
			return nil, ErrDistributionInvalid
		}
	}
	if input.Kind == DistributionKindAccountExport && !isAccountExportFileName(input.FileName) {
		return nil, ErrDistributionInvalid
	}

	recipients, users, err := s.resolveAudience(ctx, input.Audience)
	if err != nil {
		return nil, err
	}
	if len(recipients) == 0 {
		return nil, ErrDistributionAudience
	}

	audienceSnapshot := map[string]any{
		"requested":        input.Audience,
		"recipient_count": len(recipients),
	}
	preview := make([]DistributionAudienceUser, 0, min(len(users), 20))
	for _, user := range users {
		if len(preview) >= 20 {
			break
		}
		preview = append(preview, DistributionAudienceUser{ID: user.ID, Email: user.Email, Username: user.Username})
	}
	audienceSnapshot["preview"] = preview

	distribution := &Distribution{
		Title:         input.Title,
		Kind:          input.Kind,
		Content:       input.Content,
		FileName:      input.FileName,
		ContentType:   input.ContentType,
		FileSize:      int64(len(input.FileData)),
		HasAttachment: len(input.FileData) > 0,
		Metadata:      cloneDistributionMap(input.Metadata),
		Audience:      audienceSnapshot,
		CreatedBy:     input.ActorID,
		RecipientCount: int64(len(recipients)),
	}

	var attachment *DistributionAttachment
	if len(input.FileData) > 0 {
		if s.encryptor == nil {
			return nil, fmt.Errorf("distribution attachment encryptor is not configured")
		}
		encoded := base64.StdEncoding.EncodeToString(input.FileData)
		ciphertext, err := s.encryptor.Encrypt(encoded)
		if err != nil {
			return nil, fmt.Errorf("encrypt distribution attachment: %w", err)
		}
		attachment = &DistributionAttachment{
			FileName:    input.FileName,
			ContentType: input.ContentType,
			FileSize:    int64(len(input.FileData)),
			Ciphertext:  []byte(ciphertext),
		}
	}

	if err := s.repo.Create(ctx, distribution, attachment, recipients); err != nil {
		return nil, fmt.Errorf("create distribution: %w", err)
	}
	return distribution, nil
}

func (s *DistributionService) BatchCreate(ctx context.Context, inputs []CreateDistributionInput, actorID *int64) DistributionBatchResult {
	result := DistributionBatchResult{Items: make([]DistributionBatchItemResult, 0, len(inputs))}
	if len(inputs) > MaxDistributionBatchItems {
		inputs = inputs[:MaxDistributionBatchItems]
	}
	for i := range inputs {
		inputs[i].ActorID = actorID
		created, err := s.Create(ctx, inputs[i])
		item := DistributionBatchItemResult{Index: i}
		if err != nil {
			result.Failed++
			item.Error = err.Error()
		} else {
			result.Succeeded++
			item.DistributionID = created.ID
			item.RecipientCount = created.RecipientCount
		}
		result.Items = append(result.Items, item)
	}
	return result
}

func (s *DistributionService) PreviewAudience(ctx context.Context, audience DistributionAudienceInput) ([]DistributionAudienceUser, int, error) {
	_, users, err := s.resolveAudience(ctx, audience)
	if err != nil {
		return nil, 0, err
	}
	out := make([]DistributionAudienceUser, 0, min(len(users), 50))
	for _, user := range users {
		if len(out) >= 50 {
			break
		}
		out = append(out, DistributionAudienceUser{ID: user.ID, Email: user.Email, Username: user.Username})
	}
	return out, len(users), nil
}

func (s *DistributionService) ListAdmin(ctx context.Context, params pagination.PaginationParams) ([]Distribution, *pagination.PaginationResult, error) {
	return s.repo.ListAdmin(ctx, params)
}

func (s *DistributionService) ListForUser(ctx context.Context, userID int64, params pagination.PaginationParams, unreadOnly bool) ([]Distribution, *pagination.PaginationResult, error) {
	return s.repo.ListForUser(ctx, userID, params, unreadOnly)
}

func (s *DistributionService) GetForUser(ctx context.Context, id, userID int64) (*Distribution, error) {
	return s.repo.GetForUser(ctx, id, userID)
}

func (s *DistributionService) GetByID(ctx context.Context, id int64) (*Distribution, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *DistributionService) MarkRead(ctx context.Context, id, userID int64) error {
	return s.repo.MarkRead(ctx, id, userID, time.Now().UTC())
}

func (s *DistributionService) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}

func (s *DistributionService) DownloadForUser(ctx context.Context, id, userID int64) ([]byte, *DistributionAttachment, error) {
	attachment, err := s.repo.GetAttachmentForUser(ctx, id, userID)
	if err != nil {
		return nil, nil, err
	}
	data, err := s.openAttachment(attachment)
	if err != nil {
		return nil, nil, err
	}
	if err := s.repo.MarkDownloaded(ctx, id, userID, time.Now().UTC()); err != nil {
		return nil, nil, err
	}
	return data, attachment, nil
}

func (s *DistributionService) DownloadAdmin(ctx context.Context, id int64) ([]byte, *DistributionAttachment, error) {
	attachment, err := s.repo.GetAttachment(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	data, err := s.openAttachment(attachment)
	return data, attachment, err
}

func (s *DistributionService) openAttachment(attachment *DistributionAttachment) ([]byte, error) {
	if attachment == nil || len(attachment.Ciphertext) == 0 || s.encryptor == nil {
		return nil, ErrDistributionNotFound
	}
	plaintext, err := s.encryptor.Decrypt(string(attachment.Ciphertext))
	if err != nil {
		return nil, fmt.Errorf("decrypt distribution attachment: %w", err)
	}
	data, err := base64.StdEncoding.DecodeString(plaintext)
	if err != nil {
		return nil, fmt.Errorf("decode distribution attachment: %w", err)
	}
	return data, nil
}

func (s *DistributionService) resolveAudience(ctx context.Context, audience DistributionAudienceInput) ([]DistributionRecipient, []User, error) {
	recipients := make(map[int64]DistributionRecipient)
	users := make(map[int64]User)

	addUser := func(user *User, titleOverride, contentOverride string) error {
		if user == nil || user.ID <= 0 {
			return ErrUserNotFound
		}
		recipients[user.ID] = DistributionRecipient{
			UserID:          user.ID,
			TitleOverride:   strings.TrimSpace(titleOverride),
			ContentOverride: strings.TrimSpace(contentOverride),
		}
		users[user.ID] = *user
		if len(recipients) > MaxDistributionRecipients {
			return ErrDistributionTooMany
		}
		return nil
	}

	for _, id := range audience.UserIDs {
		if id <= 0 {
			continue
		}
		user, err := s.userRepo.GetByID(ctx, id)
		if err != nil {
			return nil, nil, err
		}
		if err := addUser(user, "", ""); err != nil {
			return nil, nil, err
		}
	}
	for _, email := range audience.Emails {
		email = strings.TrimSpace(email)
		if email == "" {
			continue
		}
		user, err := s.userRepo.GetByEmail(ctx, email)
		if err != nil {
			return nil, nil, err
		}
		if err := addUser(user, "", ""); err != nil {
			return nil, nil, err
		}
	}

	lineSpecs, err := ParseDistributionImportLines(audience.Lines)
	if err != nil {
		return nil, nil, err
	}
	for _, spec := range lineSpecs {
		var user *User
		if spec.UserID > 0 {
			user, err = s.userRepo.GetByID(ctx, spec.UserID)
		} else {
			user, err = s.userRepo.GetByEmail(ctx, spec.Email)
		}
		if err != nil {
			return nil, nil, err
		}
		if err := addUser(user, spec.TitleOverride, spec.ContentOverride); err != nil {
			return nil, nil, err
		}
	}

	if audience.All || audience.Filters != nil {
		filters := UserListFilters{}
		if audience.Filters != nil {
			filters.Status = strings.TrimSpace(audience.Filters.Status)
			filters.Role = strings.TrimSpace(audience.Filters.Role)
			filters.Search = strings.TrimSpace(audience.Filters.Search)
			filters.GroupName = strings.TrimSpace(audience.Filters.GroupName)
			filters.Attributes = audience.Filters.Attributes
			if activity := strings.TrimSpace(audience.Filters.Activity); activity != "" {
				normalized, ok := NormalizeUserActivityFilter(activity)
				if !ok {
					return nil, nil, ErrDistributionInvalid
				}
				filters.Search = strings.TrimSpace(filters.Search + " " + UserActivitySearchToken(normalized))
			}
		}
		includeSubscriptions := false
		filters.IncludeSubscriptions = &includeSubscriptions

		for page := 1; ; page++ {
			batch, _, listErr := s.userRepo.ListWithFilters(ctx, pagination.PaginationParams{Page: page, PageSize: 500}, filters)
			if listErr != nil {
				return nil, nil, listErr
			}
			for i := range batch {
				user := batch[i]
				if err := addUser(&user, "", ""); err != nil {
					return nil, nil, err
				}
			}
			if len(batch) < 500 {
				break
			}
		}
	}

	ids := make([]int64, 0, len(recipients))
	for id := range recipients {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })

	outRecipients := make([]DistributionRecipient, 0, len(ids))
	outUsers := make([]User, 0, len(ids))
	for _, id := range ids {
		outRecipients = append(outRecipients, recipients[id])
		outUsers = append(outUsers, users[id])
	}
	return outRecipients, outUsers, nil
}

type DistributionImportLine struct {
	UserID          int64
	Email           string
	TitleOverride   string
	ContentOverride string
}

// ParseDistributionImportLines accepts one recipient per line:
//   user-id
//   email@example.com
//   recipient | personalized title | personalized content
func ParseDistributionImportLines(raw string) ([]DistributionImportLine, error) {
	lines := strings.Split(strings.ReplaceAll(raw, "\r\n", "\n"), "\n")
	out := make([]DistributionImportLine, 0, len(lines))
	for index, rawLine := range lines {
		line := strings.TrimSpace(rawLine)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "|", 3)
		for i := range parts {
			parts[i] = strings.TrimSpace(parts[i])
		}
		if parts[0] == "" {
			return nil, fmt.Errorf("line %d: recipient is required", index+1)
		}
		spec := DistributionImportLine{}
		if id, err := strconv.ParseInt(parts[0], 10, 64); err == nil && id > 0 {
			spec.UserID = id
		} else if strings.Contains(parts[0], "@") {
			spec.Email = parts[0]
		} else {
			return nil, fmt.Errorf("line %d: recipient must be a user id or email", index+1)
		}
		if len(parts) > 1 {
			spec.TitleOverride = parts[1]
		}
		if len(parts) > 2 {
			spec.ContentOverride = parts[2]
		}
		out = append(out, spec)
	}
	return out, nil
}

func isDistributionKind(kind string) bool {
	switch kind {
	case DistributionKindText, DistributionKindMessage, DistributionKindFile, DistributionKindAccountExport:
		return true
	default:
		return false
	}
}

func isAccountExportFileName(name string) bool {
	lower := strings.ToLower(strings.TrimSpace(name))
	return strings.HasSuffix(lower, ".json") || strings.HasSuffix(lower, ".zip")
}

func sanitizeDistributionFileName(name string) string {
	name = strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(name, "\\", "_"), "/", "_"))
	if len([]rune(name)) > 255 {
		name = string([]rune(name)[:255])
	}
	return name
}

func cloneDistributionMap(input map[string]any) map[string]any {
	if input == nil {
		return map[string]any{}
	}
	data, err := json.Marshal(input)
	if err != nil {
		return map[string]any{}
	}
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		return map[string]any{}
	}
	return out
}
