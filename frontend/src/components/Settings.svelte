<script>
    import { createEventDispatcher } from 'svelte';
    import { settings } from "../stores/appStore.js";
    import { fade, fly } from 'svelte/transition';

    const dispatch = createEventDispatcher();

    function close() {
        dispatch('close');
    }

    // Закрытие по клавише ESC
    function handleKeydown(e) {
        if (e.key === 'Escape') {
            close();
        }
    }
</script>

<svelte:window on:keydown={handleKeydown}/>

<!-- transition:fly делает красивый вылет -->
<div class="settings-popup" transition:fly="{{ y: -10, duration: 200 }}">
    <div class="header">
        <h3>Настройки</h3>
        <button class="close-btn" on:click={close} aria-label="Закрыть">
            <!-- SVG иконка крестика -->
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="6" x2="6" y2="18"></line><line x1="6" y1="6" x2="18" y2="18"></line></svg>
        </button>
    </div>

    <div class="content">
        <!-- ВАШ КОНТЕНТ НАСТРОЕК (как в предыдущем ответе) -->
        
        <!-- Группа: Ресайз -->
        <div class="control-group row-group">
            <div class="label-info">
                <span class="label-text">Изменить размер</span>
                <small>Масштабировать изображение</small>
            </div>
            <label class="switch">
                <input type="checkbox" bind:checked={$settings.resize}>
                <span class="slider round"></span>
            </label>
        </div>

        <!-- Группа: Ширина -->
        <div class="control-group" class:disabled={!$settings.resize}>
            <label for="resize_to">Ширина (px)</label>
            <div class="input-wrapper">
                <input
                    type="number"
                    id="resize_to"
                    min="100"
                    max="4000"
                    disabled={!$settings.resize}
                    bind:value={$settings.resize_to}
                />
                <span class="unit">px</span>
            </div>
        </div>

        <div class="divider"></div>

        <!-- Группа: Качество -->
        <div class="control-group">
            <div class="header-row">
                <label for="webp_quality">WebP Качество</label>
                <span class="value-display">{$settings.webp_quality}%</span>
            </div>
            <div class="range-wrapper">
                <input 
                    type="range" 
                    min="10" 
                    max="100" 
                    bind:value={$settings.webp_quality} 
                    class="range-input"
                />
            </div>
        </div>
    </div>
</div>

<style>
    /* Переменные */
    :root {
        --popup-bg: #ffffff;
        --popup-border: #e5e7eb;
        --text-main: #1f2937;
        --text-secondary: #6b7280;
        --accent: #4b5563;
        --input-bg: #f9fafb;
        --shadow: 0 10px 25px -5px rgba(0, 0, 0, 0.2), 0 8px 10px -6px rgba(0, 0, 0, 0.2);
    }

    .settings-popup {
        position: absolute;
        /* Позиционирование относительно .settings-wrapper в Header.svelte */
        top: 100%; 
        right: 0;
        margin-top: 10px; /* Отступ от кнопки */
        
        width: 300px;
        background: var(--popup-bg);
        border: 1px solid var(--popup-border);
        border-radius: 12px;
        box-shadow: var(--shadow);
        z-index: 1000;
        font-family: -apple-system, sans-serif;
        color: var(--text-main);
        text-align: left; /* Сбрасываем центрирование родителя */
    }

    /* Остальные стили из предыдущего красивого варианта оставляем без изменений */
    .header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        padding: 12px 16px;
        border-bottom: 1px solid var(--popup-border);
        background: #f8f8f8;
        border-radius: 12px 12px 0 0;
    }
    
    .header h3 { margin: 0; font-size: 16px; font-weight: 600; color: #333;}
    
    .content { padding: 16px; display: flex; flex-direction: column; gap: 16px; }
    
    .close-btn { background: none; border: none; cursor: pointer; color: #888; padding: 4px; display: flex;}
    .close-btn:hover { color: #333; }

    .control-group { display: flex; flex-direction: column; gap: 6px; }
    .control-group.disabled { opacity: 0.5; pointer-events: none; }
    .row-group { flex-direction: row; justify-content: space-between; align-items: center; }
    
    .label-text { font-size: 14px; font-weight: 500; display: block; }
    small { color: #888; font-size: 11px; }
    label { font-size: 13px; font-weight: 500; color: #666; }

    input[type="number"] {
        width: 100%; padding: 8px 10px; border: 1px solid #ddd; border-radius: 6px; font-size: 14px;
        background: #f9fafb; color: #333;
    }
    .input-wrapper { position: relative; }
    .unit { position: absolute; right: 10px; top: 50%; transform: translateY(-50%); color: #999; font-size: 12px; }

    /* Switch CSS */
    .switch { position: relative; display: inline-block; width: 40px; height: 22px; }
    .switch input { opacity: 0; width: 0; height: 0; }
    .slider { position: absolute; cursor: pointer; top: 0; left: 0; right: 0; bottom: 0; background-color: #ccc; transition: .4s; border-radius: 34px; }
    .slider:before { position: absolute; content: ""; height: 16px; width: 16px; left: 3px; bottom: 3px; background-color: white; transition: .4s; border-radius: 50%; }
    input:checked + .slider { background-color: #4b5563; }
    input:checked + .slider:before { transform: translateX(18px); }

    .header-row { display: flex; justify-content: space-between; }
    .value-display { font-size: 13px; font-weight: bold; }
    .range-input { width: 100%; cursor: pointer; }
    .divider { height: 1px; background: #eee; margin: 4px 0; }
</style>
