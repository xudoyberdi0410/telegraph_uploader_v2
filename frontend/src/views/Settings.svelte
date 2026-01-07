<script>
    import { Card, Slider, Switch, TextField } from "m3-svelte";

    import { settings } from "../stores/appStore";
</script>

<div class="settings">
    <Card variant="filled">
        <label class="card-wrapper switch-settings">
            <div class="text">Изменять размер</div>
            <Switch bind:checked={$settings.resize} />
        </label>
    </Card>

    <Card variant="filled">
        <TextField
            disabled={!$settings.resize}
            label="Ширина (px)"
            bind:value={$settings.resize_to}
            type="number"
        />
    </Card>
    <Card variant="filled">
        <div class="text">Уровень сжатия</div>
        <Slider bind:value={$settings.webp_quality} />
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
