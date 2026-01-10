<script>
    import { NavCMLX, NavCMLXItem } from "m3-svelte";

    import iconHome from "@ktibow/iconset-material-symbols/home-outline";
    import iconSettings from "@ktibow/iconset-material-symbols/settings-outline";
    import iconHistory from "@ktibow/iconset-material-symbols/history";

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
            icon={iconHistory}
            text="Telegram"
            selected={appState.currentPage === "telegram"}
            onclick={() => (appState.currentPage = "telegram")}
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
            <Telegram />
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
