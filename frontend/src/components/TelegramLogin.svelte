<script>
    import {
        TelegramLoginQR,
        TelegramSubmitPassword,
    } from "../../wailsjs/go/main/App";
    import { EventsOn, EventsOff } from "../../wailsjs/runtime";
    import { Card, Button, TextField, LoadingIndicator } from "m3-svelte";

    let { onLoginSuccess } = $props();

    let step = $state("qr");
    let password = $state("");
    let error = $state("");
    let loading = $state(false);
    let qrCodeImage = $state("");

    function setStep(s) {
        step = s;
        error = "";
        loading = false;
    }

    async function startQrLogin() {
        loading = true;
        error = "";
        qrCodeImage = "";
        try {
            await TelegramLoginQR();
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

    $effect(() => {
        console.log("Setting up Telegram login event listeners");

        EventsOn("tg_qr_code", (base64Img) => {
            console.log("Event: tg_qr_code received");
            qrCodeImage = "data:image/png;base64," + base64Img;
            setStep("qr");
        });

        EventsOn("tg_request_password", () => {
            console.log("Event: tg_request_password");
            setStep("password");
        });

        EventsOn("tg_auth_success", (data) => {
            console.log("Event: tg_auth_success");
            setStep("success");
            setTimeout(() => {
                onLoginSuccess?.();
            }, 1500); // Wait a bit to show success message
        });

        EventsOn("tg_auth_error", (err) => {
            console.log("Event: tg_auth_error", err);
            error = err;
            loading = false;
        });

        // Start QR process
        startQrLogin();

        return () => {
            console.log("Cleaning up Telegram login event listeners");
            EventsOff("tg_request_password");
            EventsOff("tg_auth_success");
            EventsOff("tg_auth_error");
            EventsOff("tg_qr_code");
        };
    });
</script>

<div class="telegram-container">
    <h2>Вход в телеграм</h2>

    {#if error}
        <div class="error-banner">
            <p>{error}</p>
            <Button variant="text" onclick={startQrLogin}>Повторить</Button>
        </div>
    {/if}

    <Card variant="filled">
        <div class="auth-card-content">
            {#if step === "qr"}
                <div class="qr-container">
                    <h3>Сканируйте QR код</h3>
                    {#if qrCodeImage}
                        <img
                            src={qrCodeImage}
                            alt="Telegram QR Login"
                            class="qr-image"
                        />
                        <p class="hint">
                            Откройте Telegram на вашем телефоне > Настройки >
                            Устройства > Подключить устройство
                        </p>
                    {:else}
                        <div class="loading-state">
                            <LoadingIndicator />
                            <p>Генерация QR кода...</p>
                        </div>
                    {/if}
                </div>
            {:else if step === "password"}
                <div class="form-group">
                    <TextField
                        bind:value={password}
                        label="Пароль 2FA"
                        type="password"
                        disabled={loading}
                    />
                    <div class="actions">
                        {#if loading}
                            <LoadingIndicator />
                        {:else}
                            <Button variant="filled" onclick={handlePassword}
                                >Подтвердить</Button
                            >
                        {/if}
                    </div>
                </div>
            {:else if step === "success"}
                <div class="success-message">
                    <h3>Успешно вошли в Telegram!</h3>
                    <p>Перенаправление...</p>
                    <LoadingIndicator />
                </div>
            {/if}
        </div>
    </Card>
</div>

<style>
    .telegram-container {
        padding: 2rem;
        max-width: 480px;
        margin: 0 auto;
    }

    h2 {
        text-align: center;
        margin-bottom: 1.5rem;
        color: var(--m3c-on-surface);
    }

    .auth-card-content {
        display: flex;
        flex-direction: column;
        gap: 1rem;
        padding: 1rem;
    }

    .form-group {
        display: flex;
        flex-direction: column;
        gap: 1.5rem;
    }

    .actions {
        display: flex;
        justify-content: center;
        margin-top: 0.5rem;
    }

    .hint {
        font-size: 0.8rem;
        color: var(--m3c-on-surface-variant);
        margin-top: 0.5rem;
    }

    .error-banner {
        background-color: var(--m3c-error-container);
        color: var(--m3c-on-error-container);
        padding: 0.75rem;
        border-radius: 8px;
        margin-bottom: 1rem;
        text-align: center;
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: 0.5rem;
    }

    .error-banner p {
        margin: 0;
    }

    .success-message {
        text-align: center;
        color: var(--m3c-primary);
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: 1rem;
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
        border-radius: 8px;
        background: white; /* QR needs contrast */
        padding: 8px;
    }

    .loading-state {
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: 1rem;
    }
</style>
