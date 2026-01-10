<script>
    import { onMount, onDestroy } from "svelte";
    import {
        TelegramLogin,
        TelegramLoginQR,
        TelegramSubmitCode,
        TelegramSubmitPassword,
    } from "../../wailsjs/go/main/App";
    import { EventsOn, EventsOff } from "../../wailsjs/runtime";

    let step = $state("phone"); // phone, code, password, success
    let phone = $state("");
    let code = $state("");
    let password = $state("");
    let error = $state("");
    let loading = $state(false);
    let codeType = $state("");
    let nextType = $state("");
    let timeout = $state(0);
    let qrCodeImage = $state("");
    let isQrLogin = $state(false);
    let timerInterval;

    let cleanupEvents = [];

    function setStep(s) {
        step = s;
        error = "";
        loading = false;
        clearInterval(timerInterval);
    }

    async function handleLogin() {
        if (!phone) return;
        loading = true;
        error = "";
        codeType = "";
        // TelegramLogin returns "Process started" immediately, events follow
        try {
            await TelegramLogin(phone);
        } catch (e) {
            error = e;
            loading = false;
        }
    }

    async function handleQrLogin() {
        loading = true;
        error = "";
        isQrLogin = true;
        try {
            await TelegramLoginQR();
        } catch (e) {
            error = e;
            loading = false;
        }
    }

    async function handleCode() {
        if (!code) return;
        loading = true;
        error = "";
        try {
            await TelegramSubmitCode(code);
        } catch (e) {
            error = e;
            loading = false;
        }
    }

    async function handlePassword() {
        if (!password) return;
        loading = true;
        error = "";
        try {
            await TelegramSubmitPassword(password);
        } catch (e) {
            error = e;
            loading = false;
        }
    }

    onMount(() => {
        // Subscribe to events
        cleanupEvents.push(
            EventsOn("tg_qr_code", (base64Img) => {
                console.log("Event: tg_qr_code received");
                qrCodeImage = "data:image/png;base64," + base64Img;
                setStep("qr");
            }),
        );

        cleanupEvents.push(
            EventsOn("tg_request_code", (data) => {
                console.log("Event: tg_request_code", data);
                if (data) {
                    codeType = data.type || "";
                    nextType = data.next_type || "";
                    timeout = data.timeout || 0;

                    if (timeout > 0) {
                        clearInterval(timerInterval);
                        timerInterval = setInterval(() => {
                            if (timeout > 0) {
                                timeout--;
                            } else {
                                clearInterval(timerInterval);
                            }
                        }, 1000);
                    }
                }
                setStep("code");
            }),
        );

        cleanupEvents.push(
            EventsOn("tg_request_password", () => {
                console.log("Event: tg_request_password");
                setStep("password");
            }),
        );

        cleanupEvents.push(
            EventsOn("tg_auth_success", (data) => {
                console.log("Event: tg_auth_success");
                setStep("success");
            }),
        );

        cleanupEvents.push(
            EventsOn("tg_auth_error", (err) => {
                console.log("Event: tg_auth_error", err);
                error = err;
                loading = false;
                // Optionally reset to phone step or stay to retry
            }),
        );
    });

    onDestroy(() => {
        // In Wails runtime JS, EventsOff might need the event name.
        // If EventsOn returns a cleanup function, use it, otherwise call EventsOff manually if needed.
        // Assuming typical Wails usage where we might just want to stop listening if component unmounts.
        // But wailsjs/runtime EventsOff usually takes event name + optional callback to remove specific one.
        // For simplicity, we'll try to rely on component lifecycle not overlapping too much,
        // or effectively we might want to keep listening if this view is cached.
        // If we strictly need to unsubscribe:
        EventsOff("tg_request_code");
        EventsOff("tg_request_password");
        EventsOff("tg_auth_success");
        EventsOff("tg_auth_error");
    });
</script>

<div class="telegram-container">
    <h2>Telegram Login</h2>

    {#if error}
        <div class="error-banner">{error}</div>
    {/if}

    <div class="auth-card">
        {#if step === "phone"}
            <div class="form-group">
                <label for="phone">Phone Number</label>
                <input
                    id="phone"
                    type="text"
                    bind:value={phone}
                    placeholder="+1234567890"
                    disabled={loading}
                />
                <p class="hint">Enter your number in international format.</p>
                <button onclick={handleLogin} disabled={loading}>
                    {loading ? "Sending..." : "Send Code"}
                </button>
            </div>
            <div class="divider">OR</div>
            <button
                class="secondary-btn"
                onclick={handleQrLogin}
                disabled={loading}
            >
                Login with QR Code
            </button>
        {:else if step === "qr"}
            <div class="qr-container">
                <h3>Scan QR Code</h3>
                {#if qrCodeImage}
                    <img
                        src={qrCodeImage}
                        alt="Telegram QR Login"
                        class="qr-image"
                    />
                    <p class="hint">
                        Open Telegram on your phone > Settings > Devices > Link
                        Desktop Device
                    </p>
                {:else}
                    <p>Generating QR Code...</p>
                {/if}
                <button class="text-btn" onclick={() => setStep("phone")}
                    >Cancel</button
                >
            </div>
        {:else if step === "code"}
            <div class="form-group">
                <label for="code">Verification Code</label>
                <input
                    id="code"
                    type="text"
                    bind:value={code}
                    placeholder="12345"
                    disabled={loading}
                />
                <p class="hint">
                    {#if codeType === "app"}
                        Code sent to <strong>Telegram App</strong> on your other
                        device.
                        <br />PLEASE CHECK YOUR TELEGRAM API CHATS.
                    {:else if codeType === "sms"}
                        Code sent via <strong>SMS</strong>.
                    {:else if codeType === "call"}
                        Code sent via <strong>Phone Call</strong>.
                    {:else if codeType === "flash_call"}
                        Code sent via <strong>Flash Call</strong>.
                    {:else}
                        Code sent via <strong>{codeType}</strong>.
                    {/if}
                </p>
                {#if timeout > 0}
                    <p class="hint" style="color: #e65100;">
                        Wait <strong>{timeout}s</strong> before requesting new
                        code (via {nextType}).
                    </p>
                {/if}
                <button onclick={handleCode} disabled={loading}>
                    {loading ? "Verifying..." : "Submit Code"}
                </button>
            </div>
        {:else if step === "password"}
            <div class="form-group">
                <label for="password">2FA Password</label>
                <input
                    id="password"
                    type="password"
                    bind:value={password}
                    placeholder="Your password"
                    disabled={loading}
                />
                <button onclick={handlePassword} disabled={loading}>
                    {loading ? "Verifying..." : "Submit Password"}
                </button>
            </div>
        {:else if step === "success"}
            <div class="success-message">
                <h3>Successfully Logged In!</h3>
                <p>You can now use Telegram features.</p>
                <!-- Maybe add a button to go somewhere else or just show status -->
            </div>
        {/if}
    </div>
</div>

<style>
    .telegram-container {
        padding: 2rem;
        max-width: 480px;
        margin: 0 auto;
        font-family: sans-serif;
    }

    h2 {
        text-align: center;
        margin-bottom: 1.5rem;
    }

    .auth-card {
        background: var(--card-bg, #fff);
        padding: 2rem;
        border-radius: 12px;
        box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
    }

    .form-group {
        display: flex;
        flex-direction: column;
        gap: 1rem;
    }

    label {
        font-weight: 500;
        font-size: 0.9rem;
    }

    input {
        padding: 0.75rem;
        border: 1px solid #ddd;
        border-radius: 8px;
        font-size: 1rem;
    }

    button {
        padding: 0.75rem;
        background-color: #3390ec;
        color: white;
        border: none;
        border-radius: 8px;
        font-size: 1rem;
        cursor: pointer;
        transition: background-color 0.2s;
    }

    button:hover:not(:disabled) {
        background-color: #287dc5;
    }

    button:disabled {
        opacity: 0.7;
        cursor: not-allowed;
    }

    .hint {
        font-size: 0.8rem;
        color: #666;
        margin: -0.5rem 0 0 0;
    }

    .error-banner {
        background-color: #fee;
        color: #d32f2f;
        padding: 0.75rem;
        border-radius: 8px;
        margin-bottom: 1rem;
        text-align: center;
    }

    .success-message {
        text-align: center;
        color: #2e7d32;
    }

    .divider {
        text-align: center;
        margin: 1rem 0;
        color: #888;
        font-size: 0.8rem;
    }

    .secondary-btn {
        background-color: transparent;
        color: #3390ec;
        border: 1px solid #3390ec;
    }
    .secondary-btn:hover:not(:disabled) {
        background-color: rgba(51, 144, 236, 0.1);
    }

    .text-btn {
        background: none;
        color: #666;
        padding: 0.5rem;
        font-size: 0.9rem;
        margin-top: 1rem;
    }
    .text-btn:hover {
        color: #333;
        background: none;
        text-decoration: underline;
    }

    .qr-container {
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: 1rem;
        text-align: center;
    }

    .qr-image {
        width: 200px;
        height: 200px;
        border: 1px solid #eee;
        border-radius: 8px;
    }
</style>
