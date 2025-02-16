package v1alpha1

import (
	// uuid "github.com/google/uuid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// StockfishSpec defines the desired state of Stockfish
type StockfishSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// a valid `UCI` position command
	Position string `json:"position,omitempty"`
	// a depth to search from the given position
	Depth int `json:"depth,omitempty"`
	// how many lines to output from the search along with their evaluation
	Lines int `json:"lines,omitempty"`
}

// StockfishStatus defines the observed state of Stockfish
type StockfishStatus struct {
	// The current state of the Stockfish Analysis Job
	State string `json:"state,omitempty"`
	// The name of the Job that is running the analysis
	JobName string `json:"jobName,omitempty"`
	// The UUID of the analysis output from the Stockfish Job
	Analysis string `json:"analysis,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Stockfish is the Schema for the stockfish API
type Stockfish struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   StockfishSpec   `json:"spec,omitempty"`
	Status StockfishStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// StockfishList contains a list of Stockfish
type StockfishList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Stockfish `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Stockfish{}, &StockfishList{})
}
