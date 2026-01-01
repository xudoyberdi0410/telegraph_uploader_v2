<script>
    let { img, index, isProcessing } = $$props;
    
    import { createEventDispatcher } from 'svelte';
    const dispatch = createEventDispatcher();
</script>

<div 
    class="card" 
    class:selected={img.selected}
    draggable={!isProcessing}
    on:dragstart
    on:dragover
    on:dragend
>
    <div class="card-inner">
        <!-- Крестик удаления -->
        <button 
            class="close-btn" 
            on:click|stopPropagation={() => dispatch('removeImage')} 
            title="Убрать из списка"
        >×</button>
        
        <!-- Чекбокс -->
        <div class="checkbox-wrapper">
            <input type="checkbox" bind:checked={img.selected}>
        </div>
        
        <!-- Картинка -->
        <div class="img-wrapper">
            <img 
                src={img.thumbnailSrc} 
                alt={img.name} 
                loading="lazy" 
                decoding="async" 
                style="width: 100%; height: 100%; object-fit: contain;"
            >
        </div>
        
        <div class="name">{img.name}</div>
    </div>
</div>

<style>
    .card { 
        background: var(--card-bg); 
        border-radius: 6px; 
        aspect-ratio: 2 / 3; 
        position: relative; 
        cursor: grab;
        border: 2px solid transparent;
        transition: border-color 0.1s;
        overflow: hidden;
    }
    
    .card:active { cursor: grabbing; }
    .card:not(.selected) { opacity: 0.5; filter: grayscale(1); }
    .card.selected:hover { border-color: var(--accent); }
    
    .card-inner {
        width: 100%; 
        height: 100%;
        display: flex; 
        flex-direction: column;
    }
    
    .img-wrapper {
        flex-grow: 1;
        overflow: hidden;
        position: relative;
        background: #000;
    }
    
    .close-btn {
        position: absolute; 
        top: 6px; 
        right: 6px; 
        z-index: 10;
        background: rgba(0,0,0,0.6); 
        color: #fff; 
        border: none;
        width: 24px; 
        height: 24px; 
        border-radius: 50%;
        cursor: pointer; 
        display: flex; 
        align-items: center; 
        justify-content: center;
        opacity: 0; 
        transition: opacity 0.2s; 
        font-size: 16px;
    }
    
    .card:hover .close-btn { opacity: 1; }
    .close-btn:hover { background: #ff4444; }
    
    .name { 
        padding: 6px; 
        font-size: 0.75rem; 
        color: #888; 
        background: #252525;
        text-align: center; 
        overflow: hidden; 
        text-overflow: ellipsis; 
        white-space: nowrap; 
        flex-shrink: 0;
        pointer-events: none;
    }
    
    .checkbox-wrapper {
        position: absolute; 
        top: 6px; 
        left: 6px; 
        z-index: 10;
    }
    
    .checkbox-wrapper input { 
        width: 18px; 
        height: 18px; 
        cursor: pointer; 
        accent-color: var(--accent); 
    }
</style>
