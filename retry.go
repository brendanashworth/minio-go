/*
 * Minio Go Library for Amazon S3 Compatible Cloud Storage (C) 2015, 2016 Minio, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package minio

import (
	"net"
	"net/url"
	"strings"
	"time"
)

// MaxRetry is the maximum number of retries before stopping.
var MaxRetry = 5

// MaxJitter will randomize over the full exponential backoff time
const MaxJitter = 1.0

// NoJitter disables the use of jitter for randomizing the exponential backoff time
const NoJitter = 0.0

// newRetryTimer creates a timer with exponentially increasing delays
// until the maximum retry attempts are reached.
func (c Client) newRetryTimer(maxRetry int, unit time.Duration, cap time.Duration, jitter float64) <-chan int {
	attemptCh := make(chan int)

	// computes the exponential backoff duration according to
	// https://www.awsarchitectureblog.com/2015/03/backoff.html
	exponentialBackoffWait := func(attempt int) time.Duration {
		// normalize jitter to the range [0, 1.0]
		if jitter < NoJitter {
			jitter = NoJitter
		}
		if jitter > MaxJitter {
			jitter = MaxJitter

		}

		//sleep = random_between(0, min(cap, base * 2 ** attempt))
		sleep := unit * time.Duration(1<<uint(attempt))
		if sleep > cap {
			sleep = cap
		}
		if jitter != NoJitter {
			sleep -= time.Duration(c.random.Float64() * float64(sleep) * jitter)
		}
		return sleep
	}

	go func() {
		defer close(attemptCh)
		for i := 0; i < maxRetry; i++ {
			attemptCh <- i + 1 // Attempts start from 1.
			time.Sleep(exponentialBackoffWait(i))
		}
	}()
	return attemptCh
}

// isNetErrorRetryable - is network error retryable.
func isNetErrorRetryable(err error) bool {
	switch err.(type) {
	case *net.DNSError, *net.OpError, net.UnknownNetworkError:
		return true
	case *url.Error:
		// For a URL error, where it replies back "connection closed"
		// retry again.
		if strings.Contains(err.Error(), "Connection closed by foreign host") {
			return true
		}
	}
	return false
}

// List of S3 codes which are retryable.
var s3CodesRetryable = map[string]struct{}{
	"RequestError":          {},
	"RequestTimeout":        {},
	"Throttling":            {},
	"ThrottlingException":   {},
	"RequestLimitExceeded":  {},
	"RequestThrottled":      {},
	"InternalError":         {},
	"ExpiredToken":          {},
	"ExpiredTokenException": {},
	// Add more s3 codes here.
}

// isS3CodeRetryable - is s3 error code retryable.
func isS3CodeRetryable(s3Code string) (ok bool) {
	_, ok = s3CodesRetryable[s3Code]
	return ok
}
