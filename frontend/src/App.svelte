<script>
    import { NavCMLX, NavCMLXItem } from "m3-svelte";

    import iconHome from "@ktibow/iconset-material-symbols/home-outline";
    import iconSettings from "@ktibow/iconset-material-symbols/settings-outline";
    import iconHistory from "@ktibow/iconset-material-symbols/history";
    import iconUpload from "@ktibow/iconset-material-symbols/upload-2"

    import Home from "./views/Home.svelte";
    import Settings from "./views/Settings.svelte";
    import History from "./views/History.svelte";
    import Telegram from "./views/Telegram.svelte";

    let currentPage = "home";
</script>

<div class="app-layout">
    <NavCMLX variant="large">
        <NavCMLXItem
            variant="auto"
            icon={iconHome}
            text="Главная"
            selected={currentPage === "home"}
            onclick={() => (currentPage = "home")}
        />

        <NavCMLXItem
            variant="auto"
            icon={iconSettings}
            text="Настройки"
            selected={currentPage === "settings"}
            onclick={() => (currentPage = "settings")}
        />

        <NavCMLXItem
            variant="auto"
            icon={iconHistory}
            text="История"
            selected={currentPage === "history"}
            onclick={() => (currentPage = "history")}
        />

        <!-- <NavCMLXItem
            variant="auto"
            icon={iconUpload}
            text="Telegram"
            disabled
            selected={currentPage === "telegram"}
            onclick={() => (currentPage = "telegram")}
        /> -->
    </NavCMLX>

    <main class="content">
        {#if currentPage === "home"}
            <Home />
        {:else if currentPage === "settings"}
            <Settings />
        {:else if currentPage === "history"}
            <History />
        {:else if currentPage === "telegram"}
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
