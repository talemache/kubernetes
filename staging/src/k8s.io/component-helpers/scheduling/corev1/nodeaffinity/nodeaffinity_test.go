/*
Copyright 2020 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package nodeaffinity

import (
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

var (
	ignoreBadValue = cmpopts.IgnoreFields(field.Error{}, "BadValue")
)

func TestNodeSelectorMatch(t *testing.T) {
	tests := []struct {
		name         string
		nodeSelector v1.NodeSelector
		node         *v1.Node
		wantErr      error
		wantMatch    bool
	}{
		{
			name:      "nil node",
			wantMatch: false,
		},
		{
			name: "invalid field selector and label selector",
			nodeSelector: v1.NodeSelector{NodeSelectorTerms: []v1.NodeSelectorTerm{
				{
					MatchFields: []v1.NodeSelectorRequirement{{
						Key:      "metadata.name",
						Operator: v1.NodeSelectorOpIn,
						Values:   []string{"host_1", "host_2"},
					}},
				},
				{
					MatchExpressions: []v1.NodeSelectorRequirement{{
						Key:      "label_1",
						Operator: v1.NodeSelectorOpIn,
						Values:   []string{"label_1_val"},
					}},
					MatchFields: []v1.NodeSelectorRequirement{{
						Key:      "metadata.name",
						Operator: v1.NodeSelectorOpIn,
						Values:   []string{"host_1"},
					}},
				},
				{
					MatchExpressions: []v1.NodeSelectorRequirement{{
						Key:      "invalid key",
						Operator: v1.NodeSelectorOpIn,
						Values:   []string{"label_value"},
					}},
				},
			}},
			wantErr: field.ErrorList{
				&field.Error{
					Type:   field.ErrorTypeInvalid,
					Field:  "nodeSelectorTerms[0].matchFields[0].values",
					Detail: "must have one element",
				},
				&field.Error{
					Type:   field.ErrorTypeInvalid,
					Field:  "nodeSelectorTerms[2].matchExpressions[0]",
					Detail: `invalid label key "invalid key": name part must consist of alphanumeric characters, '-', '_' or '.', and must start and end with an alphanumeric character (e.g. 'MyName',  or 'my.name',  or '123-abc', regex used for validation is '([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9]')`,
				},
			}.ToAggregate(),
		},
		{
			name: "node matches field selector, but not labels",
			nodeSelector: v1.NodeSelector{NodeSelectorTerms: []v1.NodeSelectorTerm{
				{
					MatchExpressions: []v1.NodeSelectorRequirement{{
						Key:      "label_1",
						Operator: v1.NodeSelectorOpIn,
						Values:   []string{"label_1_val"},
					}},
					MatchFields: []v1.NodeSelectorRequirement{{
						Key:      "metadata.name",
						Operator: v1.NodeSelectorOpIn,
						Values:   []string{"host_1"},
					}},
				},
			}},
			node: &v1.Node{ObjectMeta: metav1.ObjectMeta{Name: "host_1"}},
		},
		{
			name: "node matches field selector and label selector",
			nodeSelector: v1.NodeSelector{NodeSelectorTerms: []v1.NodeSelectorTerm{
				{
					MatchExpressions: []v1.NodeSelectorRequirement{{
						Key:      "label_1",
						Operator: v1.NodeSelectorOpIn,
						Values:   []string{"label_1_val"},
					}},
					MatchFields: []v1.NodeSelectorRequirement{{
						Key:      "metadata.name",
						Operator: v1.NodeSelectorOpIn,
						Values:   []string{"host_1"},
					}},
				},
			}},
			node:      &v1.Node{ObjectMeta: metav1.ObjectMeta{Name: "host_1", Labels: map[string]string{"label_1": "label_1_val"}}},
			wantMatch: true,
		},
		{
			name: "second term matches",
			nodeSelector: v1.NodeSelector{NodeSelectorTerms: []v1.NodeSelectorTerm{
				{
					MatchExpressions: []v1.NodeSelectorRequirement{{
						Key:      "label_1",
						Operator: v1.NodeSelectorOpIn,
						Values:   []string{"label_1_val"},
					}},
				},
				{
					MatchFields: []v1.NodeSelectorRequirement{{
						Key:      "metadata.name",
						Operator: v1.NodeSelectorOpIn,
						Values:   []string{"host_1"},
					}},
				},
			}},
			node:      &v1.Node{ObjectMeta: metav1.ObjectMeta{Name: "host_1"}},
			wantMatch: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodeSelector, err := NewNodeSelector(&tt.nodeSelector)
			if diff := cmp.Diff(tt.wantErr, err, ignoreBadValue); diff != "" {
				t.Fatalf("NewNodeSelector returned unexpected error (-want,+got):\n%s", diff)
			}
			if tt.wantErr != nil {
				return
			}
			match := nodeSelector.Match(tt.node)
			if match != tt.wantMatch {
				t.Errorf("NodeSelector.Match returned %t, want %t", match, tt.wantMatch)
			}
		})
	}
}

func TestPreferredSchedulingTermsScore(t *testing.T) {
	tests := []struct {
		name           string
		prefSchedTerms []v1.PreferredSchedulingTerm
		node           *v1.Node
		wantErr        error
		wantScore      int64
	}{
		{
			name: "invalid field selector and label selector",
			prefSchedTerms: []v1.PreferredSchedulingTerm{
				{
					Weight: 1,
					Preference: v1.NodeSelectorTerm{
						MatchFields: []v1.NodeSelectorRequirement{{
							Key:      "metadata.name",
							Operator: v1.NodeSelectorOpIn,
							Values:   []string{"host_1", "host_2"},
						}},
					},
				},
				{
					Weight: 1,
					Preference: v1.NodeSelectorTerm{
						MatchExpressions: []v1.NodeSelectorRequirement{{
							Key:      "label_1",
							Operator: v1.NodeSelectorOpIn,
							Values:   []string{"label_1_val"},
						}},
						MatchFields: []v1.NodeSelectorRequirement{{
							Key:      "metadata.name",
							Operator: v1.NodeSelectorOpIn,
							Values:   []string{"host_1"},
						}},
					},
				},
				{
					Weight: 1,
					Preference: v1.NodeSelectorTerm{
						MatchExpressions: []v1.NodeSelectorRequirement{{
							Key:      "invalid key",
							Operator: v1.NodeSelectorOpIn,
							Values:   []string{"label_value"},
						}},
					},
				},
			},
			wantErr: field.ErrorList{
				&field.Error{
					Type:   field.ErrorTypeInvalid,
					Field:  "[0].matchFields[0].values",
					Detail: "must have one element",
				},
				&field.Error{
					Type:   field.ErrorTypeInvalid,
					Field:  "[2].matchExpressions[0]",
					Detail: `invalid label key "invalid key": name part must consist of alphanumeric characters, '-', '_' or '.', and must start and end with an alphanumeric character (e.g. 'MyName',  or 'my.name',  or '123-abc', regex used for validation is '([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9]')`,
				},
			}.ToAggregate(),
		},
		{
			name: "invalid field selector but no weight, error not reported",
			prefSchedTerms: []v1.PreferredSchedulingTerm{
				{
					Weight: 0,
					Preference: v1.NodeSelectorTerm{
						MatchFields: []v1.NodeSelectorRequirement{{
							Key:      "metadata.name",
							Operator: v1.NodeSelectorOpIn,
							Values:   []string{"host_1", "host_2"},
						}},
					},
				},
			},
			node: &v1.Node{ObjectMeta: metav1.ObjectMeta{Name: "host_1"}},
		},
		{
			name: "first and third term match",
			prefSchedTerms: []v1.PreferredSchedulingTerm{
				{
					Weight: 5,
					Preference: v1.NodeSelectorTerm{
						MatchFields: []v1.NodeSelectorRequirement{{
							Key:      "metadata.name",
							Operator: v1.NodeSelectorOpIn,
							Values:   []string{"host_1"},
						}},
					},
				},
				{
					Weight: 7,
					Preference: v1.NodeSelectorTerm{
						MatchExpressions: []v1.NodeSelectorRequirement{{
							Key:      "unknown_label",
							Operator: v1.NodeSelectorOpIn,
							Values:   []string{"unknown_label_val"},
						}},
					},
				},
				{
					Weight: 11,
					Preference: v1.NodeSelectorTerm{
						MatchExpressions: []v1.NodeSelectorRequirement{{
							Key:      "label_1",
							Operator: v1.NodeSelectorOpIn,
							Values:   []string{"label_1_val"},
						}},
					},
				},
			},
			node:      &v1.Node{ObjectMeta: metav1.ObjectMeta{Name: "host_1", Labels: map[string]string{"label_1": "label_1_val"}}},
			wantScore: 16,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prefSchedTerms, err := NewPreferredSchedulingTerms(tt.prefSchedTerms)
			if diff := cmp.Diff(tt.wantErr, err, ignoreBadValue); diff != "" {
				t.Fatalf("NewPreferredSchedulingTerms returned unexpected error (-want,+got):\n%s", diff)
			}
			if tt.wantErr != nil {
				return
			}
			score := prefSchedTerms.Score(tt.node)
			if score != tt.wantScore {
				t.Errorf("PreferredSchedulingTerms.Score returned %d, want %d", score, tt.wantScore)
			}
		})
	}
}

func TestNodeSelectorRequirementsAsSelector(t *testing.T) {
	matchExpressions := []v1.NodeSelectorRequirement{{
		Key:      "foo",
		Operator: v1.NodeSelectorOpIn,
		Values:   []string{"bar", "baz"},
	}}
	mustParse := func(s string) labels.Selector {
		out, e := labels.Parse(s)
		if e != nil {
			panic(e)
		}
		return out
	}
	tc := []struct {
		in       []v1.NodeSelectorRequirement
		out      labels.Selector
		wantErrs field.ErrorList
	}{
		{in: nil, out: labels.Nothing()},
		{in: []v1.NodeSelectorRequirement{}, out: labels.Nothing()},
		{
			in:  matchExpressions,
			out: mustParse("foo in (baz,bar)"),
		},
		{
			in: []v1.NodeSelectorRequirement{{
				Key:      "foo",
				Operator: v1.NodeSelectorOpExists,
				Values:   []string{"bar", "baz"},
			}},
			wantErrs: field.ErrorList{field.Invalid(field.NewPath("root").Index(0), nil, "values set must be empty for exists and does not exist")},
		},
		{
			in: []v1.NodeSelectorRequirement{{
				Key:      "foo",
				Operator: v1.NodeSelectorOpGt,
				Values:   []string{"1"},
			}},
			out: mustParse("foo>1"),
		},
		{
			in: []v1.NodeSelectorRequirement{{
				Key:      "bar",
				Operator: v1.NodeSelectorOpLt,
				Values:   []string{"7"},
			}},
			out: mustParse("bar<7"),
		},
	}

	for i, tc := range tc {
		out, err := nodeSelectorRequirementsAsSelector(tc.in, field.NewPath("root"))
		if diff := cmp.Diff(tc.wantErrs, err, ignoreBadValue); diff != "" {
			t.Errorf("nodeSelectorRequirementsAsSelector returned unexpected error (-want,+got):\n%s", diff)
		}
		if !reflect.DeepEqual(out, tc.out) {
			t.Errorf("[%v]expected:\n\t%+v\nbut got:\n\t%+v", i, tc.out, out)
		}
	}
}
