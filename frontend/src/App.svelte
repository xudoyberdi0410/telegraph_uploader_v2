<script>
  import { onMount } from 'svelte';
  // Импортируем OnFileDrop для обработки перетаскивания из ОС
  import { OnFileDrop } from '../wailsjs/runtime/runtime'; 
  import { OpenFolderDialog, OpenFilesDialog, UploadChapter, CreateTelegraphPage } from '../wailsjs/go/main/App';

  let images = [];       
  let chapterTitle = ''; 

  let isProcessing = false;
  let statusMsg = "";
  let finalUrl = "";  

  // Drag & Drop состояние
  let draggedIndex = null;

  // При старте подписываемся на события Drag&Drop из системы (из проводника Windows)
  onMount(() => {
      OnFileDrop((x, y, paths) => {
          if (isProcessing) return;
          addImagesFromPaths(paths);
      });
  });

  // --- ЛОГИКА ДОБАВЛЕНИЯ ФАЙЛОВ ---

  function addImagesFromPaths(paths) {
      if (!paths || paths.length === 0) return;
      
      const newImages = paths.map((fullPath) => {
        // Пропускаем не картинки
        if (!fullPath.match(/\.(jpg|jpeg|png|webp)$/i)) return null;

        const safePath = encodeURIComponent(fullPath);
        const fileName = fullPath.replace(/^.*[\\/]/, ''); 
        
        return {
          id: fullPath, // Уникальный ID
          name: fileName,
          thumbnailSrc: `/thumbnail/${safePath}`,
          originalPath: fullPath,
          selected: true
        };
      }).filter(Boolean);

      // Добавляем к существующим
      images = [...images, ...newImages];
      statusMsg = `Добавлено ${newImages.length} файлов`;
  }

  async function handleSelectFolder() {
    try {
      const result = await OpenFolderDialog();
      if (!result || !result.path) return;

      chapterTitle = result.title;
      // Очищаем и заменяем список (новая глава)
      images = []; 
      addImagesFromPaths(result.images);
      
    } catch (err) {
      console.error(err);
      alert("Ошибка выбора папки");
    }
  }

  async function handleSelectFiles() {
      try {
          const files = await OpenFilesDialog();
          if (files && files.length > 0) {
              addImagesFromPaths(files);
          }
      } catch (err) {
          console.error(err);
      }
  }

  // --- Drag & Drop Сортировка (Внутри приложения) ---
  
  function handleDragStart(e, index) {
      draggedIndex = index;
      e.dataTransfer.effectAllowed = 'move';
      e.dataTransfer.dropEffect = 'move';
      // Делаем элемент полупрозрачным при перетаскивании (опционально)
      e.target.style.opacity = '0.5';
  }

  function handleDragOver(e, index) {
      e.preventDefault(); // ОБЯЗАТЕЛЬНО: Разрешаем сброс
      
      // Если мы навели на другой элемент, меняем их местами
      if (draggedIndex === null || draggedIndex === index) return;

      const sourceIdx = draggedIndex;
      const targetIdx = index;

      // Меняем местами в массиве
      const newImages = [...images];
      const item = newImages[sourceIdx];
      newImages.splice(sourceIdx, 1);
      newImages.splice(targetIdx, 0, item);
      
      images = newImages;
      draggedIndex = targetIdx; // Обновляем индекс перетаскиваемого элемента
  }

  function handleDragEnd(e) {
      draggedIndex = null;
      e.target.style.opacity = '1'; // Возвращаем прозрачность
  }

  function removeImage(index) {
      images.splice(index, 1);
      images = images;
  }

  // --- Загрузка ---

  async function handleCreateArticle() {
    const selectedImages = images.filter(img => img.selected);

    if (selectedImages.length === 0) {
        alert("Список пуст или ничего не выбрано!");
        return;
    }
    if (!chapterTitle.trim()) {
        alert("Пожалуйста, введите название главы!");
        return;
    }

    const filesToUpload = selectedImages.map(img => img.originalPath);

    isProcessing = true;
    finalUrl = "";
    
    try {
        statusMsg = `Загрузка ${filesToUpload.length} изображений...`;
        const uploadRes = await UploadChapter(filesToUpload);
        
        if (!uploadRes.success) throw new Error(uploadRes.error);

        statusMsg = "Создание статьи в Telegraph...";
        const telegraphLink = await CreateTelegraphPage(chapterTitle, uploadRes.links);
        
        if (telegraphLink.startsWith("http")) {
            finalUrl = telegraphLink;
            statusMsg = "Готово!";
        } else {
            throw new Error(telegraphLink);
        }

    } catch (e) {
        statusMsg = "Ошибка: " + e.message;
    } finally {
        isProcessing = false;
    }
  }

  function copyLink() {
      navigator.clipboard.writeText(finalUrl);
      statusMsg = "Ссылка скопирована!";
  }

  function toggleAll() {
      const allSelected = images.every(i => i.selected);
      images = images.map(i => ({...i, selected: !allSelected}));
  }

  function clearAll() {
      if(confirm("Очистить список?")) {
          images = [];
          chapterTitle = "";
          statusMsg = "";
          finalUrl = "";
      }
  }
</script>

<main>
    <header>
        <div class="left">
            <div class="btn-group">
                <button class="btn secondary" on:click={handleSelectFolder} disabled={isProcessing} title="Открыть папку целиком">
                    📂 Папка
                </button>
                <button class="btn secondary" on:click={handleSelectFiles} disabled={isProcessing} title="Добавить отдельные файлы">
                    📄 Файлы
                </button>
            </div>
            
            <input 
                type="text" 
                class="title-input" 
                bind:value={chapterTitle} 
                placeholder="Название главы" 
                disabled={isProcessing}
            />
        </div>
        
        <div class="right">
            {#if images.length > 0}
                <button class="btn text-btn" on:click={clearAll} disabled={isProcessing} title="Очистить все">
                    🗑️
                </button>
                <button class="btn text-btn" on:click={toggleAll} disabled={isProcessing}>
                    ✅ Все
                </button>
                <button class="btn primary" on:click={handleCreateArticle} disabled={isProcessing}>
                    {#if isProcessing}
                        ⏳...
                    {:else}
                        📝 Создать
                    {/if}
                </button>
            {/if}
        </div>
    </header>
    
    {#if finalUrl}
        <div class="success-box">
            <span>✅ Готово:</span>
            <a href={finalUrl} target="_blank">{finalUrl}</a>
            <button class="btn small" on:click={copyLink}>Copy</button>
        </div>
    {/if}

    {#if statusMsg}
        <div class="status-bar" class:error={statusMsg.startsWith("Ошибка")}>
            {statusMsg}
        </div>
    {/if}

    <div class="grid" class:dimmed={isProcessing}>
        {#each images as img, index (img.id)}
            <!-- svelte-ignore a11y-no-static-element-interactions -->
            <div 
                class="card" 
                class:selected={img.selected}
                draggable={!isProcessing}
                on:dragstart={(e) => handleDragStart(e, index)}
                on:dragover={(e) => handleDragOver(e, index)}
                on:dragend={handleDragEnd}
            >
                <div class="card-inner">
                    <!-- Крестик удаления -->
                    <button class="close-btn" on:click|stopPropagation={() => removeImage(index)} title="Убрать из списка">×</button>

                    <!-- Чекбокс -->
                    <div class="checkbox-wrapper">
                        <input type="checkbox" bind:checked={img.selected}>
                    </div>

                    <!-- Картинка -->
                    <div class="img-wrapper">
                         <img src={img.thumbnailSrc} alt={img.name} loading="lazy">
                    </div>
                    
                    <div class="name">{img.name}</div>
                </div>
            </div>
        {/each}
        
        {#if images.length === 0}
            <div class="empty-state">
                <p>Перетащите сюда файлы или выберите папку</p>
            </div>
        {/if}
    </div>
</main>

<style>
  :root {
    --bg-color: #1a1a1a;
    --header-bg: #252525;
    --card-bg: #2a2a2a;
    --text-main: #e0e0e0;
    --accent: #4a90e2;
    --border: #333;
    --header-height: 70px;
  }

  :global(body) {
    margin: 0; background: var(--bg-color); color: var(--text-main);
    font-family: sans-serif; overflow: hidden; user-select: none;
  }

  main { display: flex; flex-direction: column; height: 100vh; }

  header {
    background: var(--header-bg); padding: 0 1.5rem;
    display: flex; justify-content: space-between; align-items: center;
    border-bottom: 1px solid var(--border); height: var(--header-height);
    flex-shrink: 0; gap: 20px;
  }

  .left, .right { display: flex; align-items: center; gap: 10px; }
  .left { flex: 1; } 

  .btn-group { display: flex; gap: 5px; }

  .title-input {
      background: #111; border: 1px solid #444; color: white;
      padding: 8px 12px; border-radius: 4px; font-size: 1rem;
      width: 100%; max-width: 350px;
  }
  .title-input:focus { outline: none; border-color: var(--accent); }

  .btn {
    padding: 0.6rem 1.2rem; border-radius: 6px; border: none;
    cursor: pointer; font-weight: bold; font-size: 0.9rem;
    transition: 0.2s;
  }
  .btn.primary { background: var(--accent); color: white; }
  .btn.primary:hover { background: #357abd; }
  .btn.primary:disabled { background: #555; cursor: not-allowed; }
  
  .btn.secondary { background: #333; color: #ddd; border: 1px solid #444; }
  .btn.secondary:hover { background: #444; }

  .btn.text-btn { background: transparent; color: #888; border: 1px solid transparent; padding: 0.6rem 0.8rem;}
  .btn.text-btn:hover { color: #fff; border-color: #444; }

  .btn.small { padding: 4px 10px; font-size: 0.8rem; margin-left: 10px; background: #2e5c2e; color: #fff;}

  .success-box {
      background: #1b3a1b; color: #4caf50; padding: 10px;
      text-align: center; border-bottom: 1px solid #2e5c2e;
      display: flex; justify-content: center; align-items: center; gap: 10px;
  }
  .success-box a { color: #80e27e; text-decoration: none; font-weight: bold; }

  .status-bar {
    background: #2a2a2a; padding: 5px; color: #aaa; text-align: center;
    border-bottom: 1px solid #444; font-size: 0.8rem;
  }
  .status-bar.error { background: #3a1b1b; color: #ff6b6b; }

  .grid {
    display: grid; 
    /* Минимальная ширина 150px, карточки будут заполнять пространство */
    grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
    gap: 15px; 
    padding: 1.5rem; 
    overflow-y: auto; 
    flex: 1;
  }
  .dimmed { opacity: 0.5; pointer-events: none; }

  /* Карточка с фиксированным соотношением сторон */
  .card { 
      background: var(--card-bg); 
      border-radius: 6px; 
      /* Вот тут магия: соотношение сторон 2 к 3 (ширина / высота) */
      aspect-ratio: 2 / 3; 
      position: relative; 
      cursor: grab;
      border: 2px solid transparent;
      transition: border-color 0.1s;
      overflow: hidden;
  }
  
  .card:active { cursor: grabbing; }
  
  /* Если не выбрано - серый */
  .card:not(.selected) { opacity: 0.5; filter: grayscale(1); }
  
  .card.selected:hover { border-color: var(--accent); }

  /* Внутренности карточки (растянуты на всю высоту) */
  .card-inner {
      width: 100%; height: 100%;
      display: flex; flex-direction: column;
  }

  .img-wrapper {
      flex-grow: 1; /* Занимает всё доступное место */
      overflow: hidden;
      position: relative;
      background: #000;
  }

  img { 
      width: 100%; height: 100%; 
      object-fit: cover; /* Картинка заполнит блок, обрезая лишнее */
      display: block; 
      pointer-events: none; /* ВАЖНО для Drag&Drop */
  }

  .name { 
      padding: 6px; 
      font-size: 0.75rem; 
      color: #888; 
      background: #252525;
      text-align: center; 
      overflow: hidden; 
      text-overflow: ellipsis; 
      white-space: nowrap; 
      flex-shrink: 0; /* Имя не сжимается */
      pointer-events: none; /* ВАЖНО для Drag&Drop */
  }

  .checkbox-wrapper {
      position: absolute; top: 6px; left: 6px; z-index: 10;
  }
  .checkbox-wrapper input { width: 18px; height: 18px; cursor: pointer; accent-color: var(--accent); }

  .close-btn {
      position: absolute; top: 6px; right: 6px; z-index: 10;
      background: rgba(0,0,0,0.6); color: #fff; border: none;
      width: 24px; height: 24px; border-radius: 50%;
      cursor: pointer; display: flex; align-items: center; justify-content: center;
      opacity: 0; transition: opacity 0.2s; font-size: 16px;
  }
  .card:hover .close-btn { opacity: 1; }
  .close-btn:hover { background: #ff4444; }

  .empty-state {
      grid-column: 1 / -1; display: flex; justify-content: center; align-items: center;
      height: 300px; color: #555; border: 2px dashed #333; border-radius: 10px;
  }
</style>
