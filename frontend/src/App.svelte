<script>
    import { NavCMLX, NavCMLXItem } from "m3-svelte";

    import iconHome from "@ktibow/iconset-material-symbols/home-outline";
    import iconSettings from "@ktibow/iconset-material-symbols/settings-outline";
    import iconHistory from "@ktibow/iconset-material-symbols/history";

    const iconTelegram = {
        body: '<path fill="currentColor" d="M16.92 5.06L4.05 10.05C3.17 10.4 3.17 10.89 3.88 11.11L7.2 12.14L14.89 7.3C15.25 7.08 15.58 7.2 15.31 7.44L9.09 13.06H9.08L9.09 13.07L8.85 16.5C9.19 16.5 9.34 16.34 9.53 16.15L11.13 14.6L14.45 17.06C15.06 17.4 15.5 17.23 15.65 16.5L17.83 6.26C18.06 5.37 17.48 4.8 16.92 5.06Z" />',
        width: 24,
        height: 24,
    };

    import Home from "./views/Home.svelte";
    import Settings from "./views/Settings.svelte";
    import History from "./views/History.svelte";
    import Telegram from "./views/Telegram.svelte";

    import { appState } from "./stores/appStore.svelte"; // Import appState

    // let currentPage = $state("home"); // Removed local state
</script>

<div class="app-layout">
    <NavCMLX variant="large">
        <NavCMLXItem
            variant="auto"
            icon={iconHome}
            text="Главная"
            selected={appState.currentPage === "home"}
            onclick={() => (appState.currentPage = "home")}
        />

        <NavCMLXItem
            variant="auto"
            icon={iconSettings}
            text="Настройки"
            selected={appState.currentPage === "settings"}
            onclick={() => (appState.currentPage = "settings")}
        />

        <NavCMLXItem
            variant="auto"
            icon={iconHistory}
            text="История"
            selected={appState.currentPage === "history"}
            onclick={() => (appState.currentPage = "history")}
        />
        <NavCMLXItem
            variant="auto"
            icon={iconTelegram}
            text="Telegram"
            selected={appState.currentPage === "telegram"}
            onclick={() => appState.navigateTo("telegram")}
        />
    </NavCMLX>

    <main class="content">
        {#if appState.currentPage === "home"}
            <Home />
        {:else if appState.currentPage === "settings"}
            <Settings />
        {:else if appState.currentPage === "history"}
            <History />
        {:else if appState.currentPage === "telegram"}
            <Telegram {...appState.pageProps} />
        {/if}
    </main>
</div>

<style>
    .app-layout {
        display: flex;
        flex-direction: row;
        height: 100vh;
        width: 100%;
        overflow: hidden;
        background-color: var(--m3c-surface);
    }

    .content {
        flex-grow: 1;
        overflow-y: auto;
        padding: 0 16px;
    }
</style>
