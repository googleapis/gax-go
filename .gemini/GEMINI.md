# Google API Extensions for Go (`gax-go`) Context

## Repository Overview
`gax-go` stands for "Google API Extensions for Go". It is a low-level support library used by the generated clients in `google-cloud-go`.

## Structure
*   `v2/`: This is the current active version of the library.
    *   **Focus Here:** All modern development happens in `v2`.
    *   `invoke.go`: Logic for invoking RPCs (the `Invoke` function).
    *   `call_option.go`: Configuration for retries (Backoff) and timeouts.
    *   `gax.go`: Core interfaces and headers.

## Architecture & Wiring
*   **Retry Loop (`v2/invoke.go`):** The `Invoke` function implements the core retry loop for operations.
    *   **Instrumentation Point:** If you need to track **retry attempts** (e.g., `http.request.resend_count`), this is the place to inspect.
*   **Configuration (`v2/call_option.go`):** Defines `CallOption` types (like `Retryer` and `Backoff`) that control the behavior of `Invoke`.

## Usage
*   This code is critical infrastructure. Bugs here affect *all* Cloud Client Libraries.
*   Changes here often require careful verification against generated code in `google-cloud-go`.
