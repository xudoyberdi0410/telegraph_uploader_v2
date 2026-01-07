<script>
    import { Card, Slider, Switch, TextField } from "m3-svelte";

    import { appState } from "../stores/appStore.svelte";

    $effect(() => {
        JSON.stringify(appState.settings);

        appState.triggerAutoSave();
    });
</script>

<div class="settings">
    <Card variant="filled">
        <label class="card-wrapper switch-settings">
            <div class="text">Изменять размер</div>
            <Switch bind:checked={appState.settings.resize} />
        </label>
    </Card>

    <Card variant="filled">
        <TextField
            disabled={!appState.settings.resize}
            label="Ширина (px)"
            bind:value={appState.settings.resize_to}
            type="number"
        />
    </Card>
    <Card variant="filled">
        <div class="text">Уровень сжатия</div>
        <Slider bind:value={appState.settings.webp_quality} />
    </Card>
</div>

<style>
    .settings {
        display: flex;
        flex-direction: column;
        gap: 1rem;
    }
    .card-wrapper {
        display: flex;
        justify-content: space-between;
        align-items: center;
    }
    .switch-settings {
        cursor: pointer;
    }
</style>
