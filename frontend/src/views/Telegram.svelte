<script>
    import { onMount } from "svelte";
    import { IsTelegramLoggedIn } from "../../wailsjs/go/main/App";
    import TelegramLogin from "../components/TelegramLogin.svelte";
    import TelegramPublish from "../components/TelegramPublish.svelte";
    import TelegramDashboard from "../components/TelegramDashboard.svelte";
    import { LoadingIndicator } from "m3-svelte";

    let { historyId = 0, articleTitle = "", titleId = 0 } = $props();

    let isLoggedIn = $state(false);
    let isLoading = $state(true);

    async function checkLoginStatus() {
        try {
            console.log("Checking Telegram login status...");
            isLoggedIn = await IsTelegramLoggedIn();
            console.log("IsLoggedIn:", isLoggedIn);
        } catch (e) {
            console.error("Failed to check login status:", e);
        } finally {
            isLoading = false;
        }
    }

    onMount(() => {
        checkLoginStatus();
    });

    function handleLoginSuccess() {
        console.log("Login success event received");
        isLoggedIn = true;
    }
</script>

<div class="view-container">
    {#if isLoading}
        <LoadingIndicator />
    {:else if isLoggedIn}
        {#if historyId && historyId !== 0}
            <TelegramPublish {historyId} {articleTitle} {titleId} />
        {:else}
            <TelegramDashboard />
        {/if}
    {:else}
        <TelegramLogin onLoginSuccess={handleLoginSuccess} />
    {/if}
</div>

<style>
    .view-container {
        width: 100%;
        height: 100%;
        /* display: flex; center/start removed to allow full stretch */
    }
</style>
