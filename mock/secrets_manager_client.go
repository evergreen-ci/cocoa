package mock

import (
	"context"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/evergreen-ci/utility"
)

// StoredSecret is a representation of a secret kept in the global secret
// storage cache.
type StoredSecret struct {
	// For the sake of simplicity, the secret ARN is synonymous with the secret
	// name.
	Name         string
	Value        string
	BinaryValue  []byte
	IsDeleted    bool
	Created      time.Time
	LastUpdated  time.Time
	LastAccessed time.Time
	Deleted      time.Time
	Tags         map[string]string
}

func newStoredSecret(in *secretsmanager.CreateSecretInput, ts time.Time) StoredSecret {
	s := StoredSecret{
		Name:         utility.FromStringPtr(in.Name),
		Value:        utility.FromStringPtr(in.SecretString),
		BinaryValue:  in.SecretBinary,
		Created:      ts,
		LastAccessed: ts,
		Tags:         newSecretsManagerTags(in.Tags),
	}
	return s
}

func exportSecretListEntry(s StoredSecret) *secretsmanager.SecretListEntry {
	return &secretsmanager.SecretListEntry{
		ARN:              utility.ToStringPtr(s.Name),
		Name:             utility.ToStringPtr(s.Name),
		CreatedDate:      utility.ToTimePtr(s.Created),
		LastAccessedDate: utility.ToTimePtr(s.LastAccessed),
		LastChangedDate:  utility.ToTimePtr(s.LastUpdated),
		DeletedDate:      utility.ToTimePtr(s.Deleted),
		Tags:             exportSecretsManagerTags(s.Tags),
	}
}

func newSecretsManagerTags(tags []*secretsmanager.Tag) map[string]string {
	converted := map[string]string{}
	for _, t := range tags {
		if t == nil {
			continue
		}
		converted[utility.FromStringPtr(t.Key)] = utility.FromStringPtr(t.Value)
	}
	return converted
}

func exportSecretsManagerTags(tags map[string]string) []*secretsmanager.Tag {
	var exported []*secretsmanager.Tag
	for k, v := range tags {
		exported = append(exported, &secretsmanager.Tag{
			Key:   utility.ToStringPtr(k),
			Value: utility.ToStringPtr(v),
		})
	}
	return exported
}

// GlobalSecretCache is a global secret storage cache that provides a simplified
// in-memory implementation of a secrets storage service. This can be used
// indirectly with the SecretsManagerClient to access and modify secrets, or
// used directly.
var GlobalSecretCache map[string]StoredSecret

func init() {
	ResetGlobalSecretCache()
}

// ResetGlobalSecretCache resets the global fake secret storage cache to an
// initialized but clean state.
func ResetGlobalSecretCache() {
	GlobalSecretCache = map[string]StoredSecret{}
}

// SecretsManagerClient provides a mock implementation of a
// cocoa.SecretsManagerClient. This makes it possible to introspect on inputs to
// the client and control the client's output. It provides some default
// implementations where possible. By default, it will issue the API calls to
// the fake GlobalSecretCache.
type SecretsManagerClient struct {
	CreateSecretInput  *secretsmanager.CreateSecretInput
	CreateSecretOutput *secretsmanager.CreateSecretOutput
	CreateSecretError  error

	GetSecretValueInput  *secretsmanager.GetSecretValueInput
	GetSecretValueOutput *secretsmanager.GetSecretValueOutput
	GetSecretValueError  error

	DescribeSecretInput  *secretsmanager.DescribeSecretInput
	DescribeSecretOutput *secretsmanager.DescribeSecretOutput
	DescribeSecretError  error

	ListSecretsInput  *secretsmanager.ListSecretsInput
	ListSecretsOutput *secretsmanager.ListSecretsOutput
	ListSecretsError  error

	UpdateSecretInput  *secretsmanager.UpdateSecretInput
	UpdateSecretOutput *secretsmanager.UpdateSecretOutput
	UpdateSecretError  error

	DeleteSecretInput  *secretsmanager.DeleteSecretInput
	DeleteSecretOutput *secretsmanager.DeleteSecretOutput
	DeleteSecretError  error

	TagResourceInput  *secretsmanager.TagResourceInput
	TagResourceOutput *secretsmanager.TagResourceOutput
	TagResourceError  error

	CloseError error
}

// CreateSecret saves the input options and returns a new mock secret. The mock
// output can be customized. By default, it will create and save a cached mock
// secret based on the input in the global secret cache.
func (c *SecretsManagerClient) CreateSecret(ctx context.Context, in *secretsmanager.CreateSecretInput) (*secretsmanager.CreateSecretOutput, error) {
	c.CreateSecretInput = in

	if c.CreateSecretOutput != nil || c.CreateSecretError != nil {
		return c.CreateSecretOutput, c.CreateSecretError
	}

	if in.Name == nil {
		return nil, awserr.New(secretsmanager.ErrCodeInvalidParameterException, "missing secret name", nil)
	}
	if in.SecretBinary != nil && in.SecretString != nil {
		return nil, awserr.New(secretsmanager.ErrCodeInvalidParameterException, "cannot specify both secret binary and secret string", nil)
	}
	if in.SecretBinary == nil && in.SecretString == nil {
		return nil, awserr.New(secretsmanager.ErrCodeInvalidParameterException, "must specify either secret binary or secret string", nil)
	}

	name := utility.FromStringPtr(in.Name)
	if s, ok := GlobalSecretCache[name]; ok && !s.IsDeleted {
		return nil, awserr.New(secretsmanager.ErrCodeResourceExistsException, "secret already exists", nil)
	}

	newSecret := newStoredSecret(in, time.Now())
	GlobalSecretCache[newSecret.Name] = newSecret

	return &secretsmanager.CreateSecretOutput{
		ARN:  utility.ToStringPtr(newSecret.Name),
		Name: utility.ToStringPtr(newSecret.Name),
	}, nil
}

// GetSecretValue saves the input options and returns an existing mock secret's
// value. The mock output can be customized. By default, it will return a cached
// mock secret if it exists in the global secret cache.
func (c *SecretsManagerClient) GetSecretValue(ctx context.Context, in *secretsmanager.GetSecretValueInput) (*secretsmanager.GetSecretValueOutput, error) {
	c.GetSecretValueInput = in

	if c.GetSecretValueOutput != nil || c.GetSecretValueError != nil {
		return c.GetSecretValueOutput, c.GetSecretValueError
	}

	if in.SecretId == nil {
		return nil, awserr.New(secretsmanager.ErrCodeInvalidParameterException, "missing secret ID", nil)
	}

	id := utility.FromStringPtr(in.SecretId)
	s := c.getSecret(id)
	if s == nil {
		return nil, awserr.New(secretsmanager.ErrCodeResourceNotFoundException, "secret not found", nil)
	}

	if s.IsDeleted {
		return nil, awserr.New(secretsmanager.ErrCodeInvalidRequestException, "secret is deleted", nil)
	}

	s.LastAccessed = time.Now()
	GlobalSecretCache[id] = *s

	return &secretsmanager.GetSecretValueOutput{
		ARN:          utility.ToStringPtr(s.Name),
		Name:         utility.ToStringPtr(s.Name),
		SecretString: utility.ToStringPtr(s.Value),
		SecretBinary: s.BinaryValue,
		CreatedDate:  utility.ToTimePtr(s.Created),
	}, nil
}

func (c *SecretsManagerClient) getSecret(id string) *StoredSecret {
	if s, ok := GlobalSecretCache[id]; ok {
		return &s
	}
	for _, s := range GlobalSecretCache {
		if s.Name == id {
			return &s
		}
	}
	return nil
}

// DescribeSecret saves the input options and returns an existing mock secret's
// metadata information. The mock output can be customized. By default, it will
// return information about the cached mock secret if it exists in the global
// secret cache.
func (c *SecretsManagerClient) DescribeSecret(ctx context.Context, in *secretsmanager.DescribeSecretInput) (*secretsmanager.DescribeSecretOutput, error) {
	c.DescribeSecretInput = in

	if c.DescribeSecretOutput != nil || c.DescribeSecretError != nil {
		return c.DescribeSecretOutput, c.DescribeSecretError
	}

	if in.SecretId == nil {
		return nil, awserr.New(secretsmanager.ErrCodeInvalidParameterException, "missing secret ID", nil)
	}

	s, ok := GlobalSecretCache[utility.FromStringPtr(in.SecretId)]
	if !ok {
		return nil, awserr.New(secretsmanager.ErrCodeResourceNotFoundException, "secret not found", nil)
	}

	return &secretsmanager.DescribeSecretOutput{
		ARN:              utility.ToStringPtr(s.Name),
		Name:             utility.ToStringPtr(s.Name),
		CreatedDate:      utility.ToTimePtr(s.Created),
		LastAccessedDate: utility.ToTimePtr(s.LastAccessed),
		LastChangedDate:  utility.ToTimePtr(s.LastUpdated),
		DeletedDate:      utility.ToTimePtr(s.Deleted),
		Tags:             exportSecretsManagerTags(s.Tags),
	}, nil
}

// ListSecrets saves the input options and returns all matching mock secrets'
// metadata information. The mock output can be customized. By default, it will
// return any matching cached mock secrets in the global secret cache.
func (c *SecretsManagerClient) ListSecrets(ctx context.Context, in *secretsmanager.ListSecretsInput) (*secretsmanager.ListSecretsOutput, error) {
	c.ListSecretsInput = in

	if c.ListSecretsOutput != nil || c.ListSecretsError != nil {
		return c.ListSecretsOutput, c.ListSecretsError
	}

	// Get the subset of secrets that match each and every one of the filters.
	var matchingAllFilters map[string]StoredSecret
	if len(in.Filters) != 0 {
		for _, f := range in.Filters {
			if f == nil {
				continue
			}

			matchingValues := map[string]StoredSecret{}
			// kim: TODO: remove
			// for _, v := range filter.Values {
			//     if v == nil {
			//         continue
			//     }
			//
			//     switch utility.FromStringPtr(filter.Key) {
			//     case "name":
			//         for arn, s := range c.secretsMatchingAnyNameValue(utility.FromStringPtr(v)) {
			//             matchingValues[arn] = GlobalSecretCache[arn]
			//         }
			//     case "tag-key":
			//         for _, arn := range c.arnsMatchingTagKeyPrefix(utility.FromStringPtr(v)) {
			//             matchingValues[arn] = GlobalSecretCache[arn]
			//         }
			//     case "tag-value":
			//         for _, arn := range c.arnsMatchingTagKeyPrefix(utility.FromStringPtr(v)) {
			//             matchingValues[arn] = GlobalSecretCache[arn]
			//         }
			//     }
			// }
			switch utility.FromStringPtr(f.Key) {
			case "name":
				// kim: TODO: remove
				// matchingValues = c.getMergedSet(matchingValues, c.secretsMatchingAnyNameValue(utility.FromStringPtrSlice(f.Values)))
				matchingValues = c.secretsMatchingAnyNameValue(utility.FromStringPtrSlice(f.Values))
				// This could support other filter keys, but it's not worth it
				// unless the need arises.
			default:
				return nil, awserr.New(secretsmanager.ErrCodeInvalidParameterException, "unsupported filter", nil)
			}

			if matchingAllFilters == nil {
				// Initialize the candidate set of matching secrets.
				matchingAllFilters = matchingValues
			} else {
				// Each matching secret must match all the given filters.
				matchingAllFilters = c.getSetIntersection(matchingAllFilters, matchingValues)
			}
		}
	} else {
		// If no filters are given, return all the secrets.
		matchingAllFilters = GlobalSecretCache
	}

	var converted []*secretsmanager.SecretListEntry
	for _, s := range matchingAllFilters {
		converted = append(converted, exportSecretListEntry(s))
	}

	return &secretsmanager.ListSecretsOutput{
		SecretList: converted,
	}, nil
}

func (c *SecretsManagerClient) getSetIntersection(a, b map[string]StoredSecret) map[string]StoredSecret {
	intersection := map[string]StoredSecret{}
	for id, s := range a {
		if _, ok := b[id]; ok {
			intersection[id] = s
		}
	}
	return intersection
}

// kim: TODO: remove
// func (c *SecretsManagerClient) getMergedSet(a, b map[string]StoredSecret) map[string]StoredSecret {
//     merged := map[string]StoredSecret{}
//     for _, s := range a {
//         merged[s.ARN.String()] = s
//     }
//     for _, s := range b {
//         merged[s.ARN.String()] = s
//     }
//     return merged
// }

// secretsMatchingAnyNameValue returns the ARNs of all secret names that match
// any of the given values. If the value begins with a "!", the match is
// negated.
func (c *SecretsManagerClient) secretsMatchingAnyNameValue(vals []string) map[string]StoredSecret {
	secrets := map[string]StoredSecret{}
	for _, s := range GlobalSecretCache {
		if s.IsDeleted {
			continue
		}

		for _, val := range vals {
			if strings.HasPrefix(val, "!") && s.Name != val[1:] {
				secrets[s.Name] = s
			}
			if !strings.HasPrefix(val, "!") && s.Name == val {
				secrets[s.Name] = s
			}
		}
	}
	return secrets
}

// kim: TODO: remove
// arnsMatchingNameValue returns the ARNs of all secret names that match the
// given value. If the value begins with a "!", the match is negated.
// func (c *SecretsManagerClient) arnsMatchingNameValue(val string) []string {
//     var arns []string
//     for _, s := range GlobalSecretCache {
//         if s.IsDeleted {
//             continue
//         }
//
//         if strings.HasPrefix(val, "!") && s.Name != val[1:] {
//             arns = append(arns, s.ARN.String())
//         }
//         if !strings.HasPrefix(val, "!") && s.Name == val {
//             arns = append(arns, s.ARN.String())
//         }
//     }
//     return arns
// }

// kim: TODO: remove
// arnsMatchingTagKeyPrefix returns the ARNs of all secrets containing tag keys
// that match the given prefix. If the prefix begins with a "!", the match is
// negated.
// func (c *SecretsManagerClient) arnsMatchingTagKeyPrefix(prefix string) []string {
//     var arns []string
//     for _, s := range GlobalSecretCache {
//         if s.IsDeleted {
//             continue
//         }
//
//         for k := range s.Tags {
//             if strings.HasPrefix(prefix, "!") && !strings.HasPrefix(k, prefix[1:]) {
//                 arns = append(arns, s.ARN.String())
//             }
//             if !strings.HasPrefix(prefix, "!") && strings.HasPrefix(k, prefix) {
//                 arns = append(arns, s.ARN.String())
//             }
//         }
//     }
//     return arns
// }

// kim: TODO: remove
// arnsMatchingTagValuePrefix returns the ARNs of all secrets containing tag
// values that match the given prefix. If the prefix begins with a "!", the
// match is negated.
// func (c *SecretsManagerClient) arnsMatchingTagValuePrefix(prefix string) []string {
//     var arns []string
//     for _, s := range GlobalSecretCache {
//         if s.IsDeleted {
//             continue
//         }
//
//         for _, v := range s.Tags {
//             if strings.HasPrefix(prefix, "!") && !strings.HasPrefix(v, prefix[1:]) {
//                 arns = append(arns, s.ARN.String())
//             }
//             if !strings.HasPrefix(prefix, "!") && strings.HasPrefix(v, prefix) {
//                 arns = append(arns, s.ARN.String())
//             }
//         }
//     }
//     return arns
// }

// UpdateSecretValue saves the input options and returns an updated mock secret
// value. The mock output can be customized. By default, it will update a cached
// mock secret if it exists in the global secret cache.
func (c *SecretsManagerClient) UpdateSecretValue(ctx context.Context, in *secretsmanager.UpdateSecretInput) (*secretsmanager.UpdateSecretOutput, error) {
	c.UpdateSecretInput = in

	if c.UpdateSecretOutput != nil || c.UpdateSecretError != nil {
		return c.UpdateSecretOutput, c.UpdateSecretError
	}

	if in.SecretId == nil {
		return nil, awserr.New(secretsmanager.ErrCodeInvalidParameterException, "missing secret ID", nil)
	}
	if in.SecretBinary != nil && in.SecretString != nil {
		return nil, awserr.New(secretsmanager.ErrCodeInvalidParameterException, "cannot specify both secret binary and secret string", nil)
	}
	if in.SecretBinary == nil && in.SecretString == nil {
		return nil, awserr.New(secretsmanager.ErrCodeInvalidParameterException, "must specify either secret binary or secret string", nil)
	}

	id := utility.FromStringPtr(in.SecretId)
	s, ok := GlobalSecretCache[id]
	if !ok {
		return nil, awserr.New(secretsmanager.ErrCodeResourceNotFoundException, "secret not found", nil)
	}

	if s.IsDeleted {
		return nil, awserr.New(secretsmanager.ErrCodeInvalidRequestException, "secret is deleted", nil)
	}

	if in.SecretBinary != nil {
		s.BinaryValue = in.SecretBinary
	}
	if in.SecretString != nil {
		s.Value = *in.SecretString
	}

	ts := time.Now()
	s.LastAccessed = ts
	s.LastUpdated = ts

	GlobalSecretCache[id] = s

	return &secretsmanager.UpdateSecretOutput{
		ARN:  utility.ToStringPtr(s.Name),
		Name: utility.ToStringPtr(s.Name),
	}, nil
}

// DeleteSecret saves the input options and deletes an existing mock secret. The
// mock output can be customized. By default, it will delete a cached mock
// secret if it exists.
func (c *SecretsManagerClient) DeleteSecret(ctx context.Context, in *secretsmanager.DeleteSecretInput) (*secretsmanager.DeleteSecretOutput, error) {
	c.DeleteSecretInput = in

	if c.DeleteSecretOutput != nil || c.DeleteSecretError != nil {
		return c.DeleteSecretOutput, c.DeleteSecretError
	}

	if in.SecretId == nil {
		return nil, awserr.New(secretsmanager.ErrCodeInvalidParameterException, "missing secret ID", nil)
	}

	if utility.FromBoolPtr(in.ForceDeleteWithoutRecovery) && in.RecoveryWindowInDays != nil {
		return nil, awserr.New(secretsmanager.ErrCodeInvalidParameterException, "cannot force delete without recovery and also schedule a recovery window", nil)
	}

	window := int(utility.FromInt64Ptr(in.RecoveryWindowInDays))
	if in.RecoveryWindowInDays != nil && (window < 7 || window > 30) {
		return nil, awserr.New(secretsmanager.ErrCodeInvalidParameterException, "recovery window must be between 7 and 30 days", nil)
	}
	if window == 0 {
		window = 30
	}

	id := utility.FromStringPtr(in.SecretId)
	s, ok := GlobalSecretCache[id]
	if !utility.FromBoolPtr(in.ForceDeleteWithoutRecovery) && !ok {
		return nil, awserr.New(secretsmanager.ErrCodeResourceNotFoundException, "secret not found", nil)
	}

	ts := time.Now()
	s.LastAccessed = ts
	s.LastUpdated = ts
	if !utility.FromBoolPtr(in.ForceDeleteWithoutRecovery) {
		s.Deleted = ts.AddDate(0, 0, window)
	}
	s.IsDeleted = true
	GlobalSecretCache[id] = s

	return &secretsmanager.DeleteSecretOutput{
		ARN:          utility.ToStringPtr(s.Name),
		Name:         utility.ToStringPtr(s.Name),
		DeletionDate: utility.ToTimePtr(s.Deleted),
	}, nil
}

// TagResource saves the input options and tags an existing mock secret. The
// mock output can be customized. By default, it will tag the cached mock
// secret if it exists.
func (c *SecretsManagerClient) TagResource(ctx context.Context, in *secretsmanager.TagResourceInput) (*secretsmanager.TagResourceOutput, error) {
	c.TagResourceInput = in

	if c.TagResourceOutput != nil || c.TagResourceError != nil {
		return c.TagResourceOutput, c.TagResourceError
	}

	id := utility.FromStringPtr(in.SecretId)

	s, ok := GlobalSecretCache[id]
	if !ok {
		return nil, awserr.New(secretsmanager.ErrCodeResourceExistsException, "secret not found", nil)
	}

	if s.IsDeleted {
		return nil, awserr.New(secretsmanager.ErrCodeInvalidRequestException, "secret is deleted", nil)
	}

	for k, v := range newSecretsManagerTags(in.Tags) {
		s.Tags[k] = v
	}
	return &secretsmanager.TagResourceOutput{}, nil
}

// Close closes the mock client. The mock output can be customized. By default,
// it is a no-op that returns no error.
func (c *SecretsManagerClient) Close(ctx context.Context) error {
	if c.CloseError != nil {
		return c.CloseError
	}
	return nil
}
