// Copyright 2023 The Atlas Operator Authors.
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

package v1alpha1

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type (
	//+kubebuilder:object:root=true
	//
	// AtlasSchemaList contains a list of AtlasSchema
	AtlasSchemaList struct {
		metav1.TypeMeta `json:",inline"`
		metav1.ListMeta `json:"metadata,omitempty"`

		Items []AtlasSchema `json:"items"`
	}
	//+kubebuilder:object:root=true
	//+kubebuilder:subresource:status
	//
	// AtlasSchema is the Schema for the atlasschemas API
	// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
	// +kubebuilder:printcolumn:name="Reason",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].reason`
	AtlasSchema struct {
		metav1.TypeMeta   `json:",inline"`
		metav1.ObjectMeta `json:"metadata,omitempty"`

		Spec   AtlasSchemaSpec   `json:"spec,omitempty"`
		Status AtlasSchemaStatus `json:"status,omitempty"`
	}
	// AtlasSchemaStatus defines the observed state of AtlasSchema
	AtlasSchemaStatus struct {
		// Conditions represent the latest available observations of an object's state.
		Conditions []metav1.Condition `json:"conditions,omitempty"`
		// ObservedHash is the hash of the most recently applied schema.
		ObservedHash string `json:"observed_hash"`
		// LastApplied is the unix timestamp of the most recent successful schema apply operation.
		LastApplied int64 `json:"last_applied"`
		// PlanURL is the URL of the schema plan to apply.
		// +optional
		PlanURL string `json:"planURL"`
		// PlanLink is the link to the schema plan on the Atlas Cloud.
		// +optional
		PlanLink string `json:"planLink"`
	}
	// AtlasSchemaSpec defines the desired state of AtlasSchema
	AtlasSchemaSpec struct {
		TargetSpec `json:",inline"`
		// Desired Schema of the target.
		Schema Schema `json:"schema,omitempty"`
		// Cloud defines the Atlas Cloud configuration.
		Cloud Cloud `json:"cloud,omitempty"`
		// +optional
		// DevURL is the URL of the database to use for normalization and calculations.
		// If not specified, the operator will spin up a temporary database container to use for these operations.
		DevURL string `json:"devURL"`
		// DevLabels is a set of labels to apply to the temporary database container.
		// +optional
		DevLabels map[string]string `json:"devLabels,omitempty"`
		// DevAnnotations is a set of annotations to apply to the temporary database container.
		// +optional
		DevAnnotations map[string]string `json:"devAnnotations,omitempty"`
		// DevURLFrom is a reference to a secret containing the URL of the database to use for normalization and calculations.
		// +optional
		DevURLFrom Secret `json:"devURLFrom,omitempty"`
		// Exclude a list of glob patterns used to filter existing resources being taken into account.
		Exclude []string `json:"exclude,omitempty"`
		// TxMode defines the transaction mode to use when applying the schema.
		// +kubebuilder:default=file
		TxMode TransactionMode `json:"txMode,omitempty"`
		// Policy defines the policies to apply when managing the schema change lifecycle.
		Policy *Policy `json:"policy,omitempty"`
		// The names of the schemas (named databases) on the target database to be managed.
		Schemas []string `json:"schemas,omitempty"`
	}
	// Schema defines the desired state of the target database schema in plain SQL or HCL.
	Schema struct {
		SQL string `json:"sql,omitempty"`
		HCL string `json:"hcl,omitempty"`
		URL string `json:"url,omitempty"`

		ConfigMapKeyRef *corev1.ConfigMapKeySelector `json:"configMapKeyRef,omitempty"`
	}
	// Policy defines the policies to apply when managing the schema change lifecycle.
	Policy struct {
		Lint *Lint `json:"lint,omitempty"`
		Diff *Diff `json:"diff,omitempty"`
	}
	// Lint defines the linting policies to apply before applying the schema.
	Lint struct {
		Destructive *CheckConfig `json:"destructive,omitempty"`
		// Review defines the review policy to apply after linting the schema changes.
		// +kubebuilder:default=ERROR
		Review LintReview `json:"review,omitempty"`
	}
	// CheckConfig defines the configuration of a linting check.
	CheckConfig struct {
		Error bool `json:"error,omitempty"`
	}
	// Diff defines the diff policies to apply when planning schema changes.
	Diff struct {
		ConcurrentIndex *ConcurrentIndex `json:"concurrent_index,omitempty"`
		Skip            *SkipChanges     `json:"skip,omitempty"`
	}
	// SkipChanges represents the skip changes policy.
	SkipChanges struct {
		// +optional
		AddSchema bool `json:"add_schema,omitempty"`
		// +optional
		DropSchema bool `json:"drop_schema,omitempty"`
		// +optional
		ModifySchema bool `json:"modify_schema,omitempty"`
		// +optional
		AddTable bool `json:"add_table,omitempty"`
		// +optional
		DropTable bool `json:"drop_table,omitempty"`
		// +optional
		ModifyTable bool `json:"modify_table,omitempty"`
		// +optional
		AddColumn bool `json:"add_column,omitempty"`
		// +optional
		DropColumn bool `json:"drop_column,omitempty"`
		// +optional
		ModifyColumn bool `json:"modify_column,omitempty"`
		// +optional
		AddIndex bool `json:"add_index,omitempty"`
		// +optional
		DropIndex bool `json:"drop_index,omitempty"`
		// +optional
		ModifyIndex bool `json:"modify_index,omitempty"`
		// +optional
		AddForeignKey bool `json:"add_foreign_key,omitempty"`
		// +optional
		DropForeignKey bool `json:"drop_foreign_key,omitempty"`
		// +optional
		ModifyForeignKey bool `json:"modify_foreign_key,omitempty"`
	}
	ConcurrentIndex struct {
		// +optional
		Create bool `json:"create,omitempty"`
		// +optional
		Drop bool `json:"drop,omitempty"`
	}
	// TransactionMode
	// +kubebuilder:validation:Enum=file;all;none
	TransactionMode string
	// LintReview defines the review policies to apply after linting the schema.
	// +kubebuilder:validation:Enum=ALWAYS;WARNING;ERROR
	LintReview string
)

// LintReview values.
const (
	LintReviewAlways  LintReview = "ALWAYS"
	LintReviewWarning LintReview = "WARNING"
	LintReviewError   LintReview = "ERROR"
)

func init() {
	SchemeBuilder.Register(&AtlasSchema{}, &AtlasSchemaList{})
}

// NamespacedName returns the namespaced name of the object.
func (s *AtlasSchema) NamespacedName() types.NamespacedName {
	return types.NamespacedName{
		Name:      s.Name,
		Namespace: s.Namespace,
	}
}

// IsReady returns true if the ready condition is true.
func (m *AtlasSchema) IsReady() bool {
	return meta.IsStatusConditionTrue(m.Status.Conditions, readyCond)
}

// IsHashModified returns true if the hash is different from the observed hash.
func (sc *AtlasSchema) IsHashModified(hash string) bool {
	return hash != sc.Status.ObservedHash
}

// SetReady sets the Ready condition to true
func (sc *AtlasSchema) SetReady(status AtlasSchemaStatus, report any) {
	var msg string
	if report != nil {
		if j, err := json.Marshal(report); err != nil {
			msg = fmt.Sprintf("Error marshalling apply response: %v", err)
		} else {
			msg = fmt.Sprintf("The schema has been applied successfully. Apply response: %s", j)
		}
	} else {
		msg = "The schema has been applied successfully."
	}
	sc.Status = status
	meta.SetStatusCondition(&sc.Status.Conditions, metav1.Condition{
		Type:    readyCond,
		Status:  metav1.ConditionTrue,
		Reason:  "Applied",
		Message: msg,
	})
}

// SetNotReady sets the Ready condition to false
// with the given reason and message.
func (sc *AtlasSchema) SetNotReady(reason, msg string) {
	meta.SetStatusCondition(&sc.Status.Conditions, metav1.Condition{
		Type:    readyCond,
		Status:  metav1.ConditionFalse,
		Reason:  reason,
		Message: msg,
	})
}

// Schema reader types (URL schemes).
const (
	SchemaTypeAtlas = "atlas"
	SchemaTypeFile  = "file"
)

// Desired returns the desired schema of the AtlasSchema.
func (s Schema) DesiredState(ctx context.Context, r client.Reader, ns string) (*url.URL, []byte, error) {
	switch ref := s.ConfigMapKeyRef; {
	case ref != nil:
		val := &corev1.ConfigMap{}
		err := r.Get(ctx, types.NamespacedName{Name: ref.Name, Namespace: ns}, val)
		if err != nil {
			return nil, nil, err
		}
		// Guess the schema file format based on the key's extension.
		ext := strings.ToLower(filepath.Ext(ref.Key))
		switch desired, ok := val.Data[ref.Key]; {
		case !ok:
			return nil, nil, fmt.Errorf("configmaps %s/%s does not contain key %q", ns, ref.Name, ref.Key)
		case ext == ".hcl" || ext == ".sql":
			return &url.URL{Scheme: SchemaTypeFile, Path: "schema" + ext}, []byte(desired), nil
		default:
			return nil, nil, fmt.Errorf("configmaps key %q must be ending with .sql or .hcl, received %q", ref.Key, ext)
		}
	case s.HCL != "":
		return &url.URL{Scheme: SchemaTypeFile, Path: "schema.hcl"}, []byte(s.HCL), nil
	case s.SQL != "":
		return &url.URL{Scheme: SchemaTypeFile, Path: "schema.sql"}, []byte(s.SQL), nil
	case s.URL != "":
		u, err := url.Parse(s.URL)
		if err == nil && u.Scheme != SchemaTypeAtlas {
			return nil, nil, fmt.Errorf("unsupported URL schema %q", u.Scheme)
		}
		return u, nil, err
	}
	return nil, nil, fmt.Errorf("no desired state specified")
}
