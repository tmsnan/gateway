// Copyright Envoy Gateway Authors
// SPDX-License-Identifier: Apache-2.0
// The full text of the Apache license is available in the LICENSE file at
// the root of the repo.

package v1alpha1

import "github.com/golang/protobuf/ptypes/duration"

// RetryStrategy  defines the retry strategy to be applied.
type RetryStrategy struct {
	// Type decides the type of RetryStrategy protocol policy.
	// Valid ProtocolType values are
	// "Http",
	// "Grpc",
	//
	Type ProtocolType `json:"type,omitempty"`

	Http *HttpRetry `json:"http,omitempty"`

	Grpc *GrpcRetry `json:"grpc,omitempty"`

	NumRetries int              `json:"numRetries,omitempty"`
	PerRetry   PerRetryPolicy   `json:"perRetry,omitempty"`
	RetryLimit RetryLimitPolicy `json:"retryLimit,omitempty"`
}

// LoadBalancerType specifies the types of LoadBalancer.
// +kubebuilder:validation:Enum=ConsistentHash;LeastRequest;Random;RoundRobin
type ProtocolType string

type HttpRetry struct {
	RetryOn              RetryOn              `json:"retryOn,omitempty"`
	RetriableStatusCodes RetriableStatusCodes `json:"retriableStatusCodes,omitempty"`
}

type GrpcRetry struct {
	RetryOn RetryOn `json:"retryOn,omitempty"`
}

type RetryOn string
type RetriableStatusCodes []int

type PerRetryPolicy struct {
	Timeout     duration.Duration `json:"timeout,omitempty"`
	IdleTimeout duration.Duration `json:"idleTimeout,omitempty"`
	BackOff     BackOffPolicy     `json:"backOff,omitempty"`
}

type BackOffPolicy struct {
	BaseInterval duration.Duration `json:"baseInterval,omitempty"`
	MaxInterval  duration.Duration `json:"maxInterval,omitempty"`
}

type RetryLimitPolicy struct {
	// Valid RetryLimitType values are
	// "Http",
	// "Grpc",
	Type        RetryLimitType    `json:"type,omitempty"`
	Static      StaticPolicy      `json:"static,omitempty"`
	RetryBudget RetryBudgetPolicy `json:"retryBudget,omitempty"`
}
type RetryLimitType string

type StaticPolicy struct {
	MaxParallel int `json:"maxParallel,omitempty"`
}

type RetryBudgetPolicy struct {
	ActiveRequestPercent int `json:"activeRequestPercent,omitempty"`
	MinConcurrent        int `json:"minConcurrent,omitempty"`
}
