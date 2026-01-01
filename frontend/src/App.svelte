<script>
  import { OpenFolderDialog, UploadChapter, CreateTelegraphPage } from '../wailsjs/go/main/App';

  let images = [];       
  let rawFilePaths = []; 
  let chapterTitle = ''; // Теперь это привязано к Input

  let isProcessing = false;
  let statusMsg = "";
  let finalUrl = "";  

  async function handleSelectFolder() {
    try {
      const result = await OpenFolderDialog();
      if (!result || !result.path) return;

      // Автоматически подставляем имя папки, но даем редактировать
      chapterTitle = result.title;
      rawFilePaths = result.images;
      
      images = result.images.map(fullPath => {
        const safePath = encodeURIComponent(fullPath);
        const fileName = fullPath.replace(/^.*[\\/]/, '');
        return {
          name: fileName,
          thumbnailSrc: `/thumbnail/${safePath}`,
        };
      });

      finalUrl = "";
      statusMsg = "";
    } catch (err) {
      console.error(err);
      alert("Ошибка выбора папки");
    }
  }

  async function handleCreateArticle() {
    if (rawFilePaths.length === 0) return;
    if (!chapterTitle.trim()) {
        alert("Пожалуйста, введите название главы!");
        return;
    }

    isProcessing = true;
    finalUrl = "";
    
    try {
        // ШАГ 1: Загрузка на R2
        statusMsg = "1/2: Сжатие и загрузка картинок...";
        const uploadRes = await UploadChapter(rawFilePaths);
        
        if (!uploadRes.success) {
            throw new Error(uploadRes.error);
        }

        // ШАГ 2: Создание статьи
        statusMsg = "2/2: Создание статьи в Telegraph...";
        // Передаем название и полученные ссылки
        const telegraphLink = await CreateTelegraphPage(chapterTitle, uploadRes.links);
        
        // Проверяем, вернулась ссылка или сообщение об ошибке
        if (telegraphLink.startsWith("http")) {
            finalUrl = telegraphLink;
            statusMsg = "Готово!";
            console.log("Article created:", finalUrl);
        } else {
            throw new Error(telegraphLink); // Сообщение об ошибке от Go
        }

    } catch (e) {
        statusMsg = "Ошибка: " + e.message;
    } finally {
        isProcessing = false;
    }
  }

  function copyLink() {
      navigator.clipboard.writeText(finalUrl);
      alert("Ссылка скопирована!");
  }
</script>

<main>
    <header>
        <div class="left">
            <button class="btn secondary" on:click={handleSelectFolder} disabled={isProcessing}>
                📂 Папка
            </button>
            
            <!-- Поле ввода названия -->
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
                <button class="btn primary" on:click={handleCreateArticle} disabled={isProcessing}>
                    {#if isProcessing}
                        ⏳ Работы...
                    {:else}
                        📝 Создать статью
                    {/if}
                </button>
            {/if}
        </div>
    </header>
    
    <!-- Результат -->
    {#if finalUrl}
        <div class="success-box">
            <span>✅ Статья создана:</span>
            <a href={finalUrl} target="_blank">{finalUrl}</a>
            <button class="btn small" on:click={copyLink}>Copy</button>
        </div>
    {/if}

    <!-- Статус бар -->
    {#if statusMsg}
        <div class="status-bar" class:error={statusMsg.startsWith("Ошибка")}>
            {statusMsg}
        </div>
    {/if}

    <div class="grid" class:dimmed={isProcessing}>
        {#each images as img}
            <div class="card">
                <img src={img.thumbnailSrc} alt={img.name} loading="lazy">
                <div class="name">{img.name}</div>
            </div>
        {/each}
        
        {#if images.length === 0}
            <div class="empty-state">
                <p>Выберите папку, введите название и нажмите "Создать статью"</p>
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
    font-family: sans-serif; overflow: hidden;
  }

  main { display: flex; flex-direction: column; height: 100vh; }

  header {
    background: var(--header-bg); padding: 0 1.5rem;
    display: flex; justify-content: space-between; align-items: center;
    border-bottom: 1px solid var(--border); height: var(--header-height);
    flex-shrink: 0; gap: 20px;
  }

  .left, .right { display: flex; align-items: center; gap: 15px; }
  .left { flex: 1; } /* Растягиваем левую часть для инпута */

  .title-input {
      background: #111; border: 1px solid #444; color: white;
      padding: 8px 12px; border-radius: 4px; font-size: 1rem;
      width: 100%; max-width: 400px;
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
  
  .btn.small { padding: 4px 10px; font-size: 0.8rem; margin-left: 10px; }

  .success-box {
      background: #1b3a1b; color: #4caf50; padding: 15px;
      text-align: center; border-bottom: 1px solid #2e5c2e;
      display: flex; justify-content: center; align-items: center; gap: 10px;
  }
  .success-box a { color: #80e27e; text-decoration: none; font-weight: bold; }
  .success-box a:hover { text-decoration: underline; }

  .status-bar {
    background: #2a2a2a; padding: 8px; color: #fff; text-align: center;
    border-bottom: 1px solid #444; font-size: 0.9rem;
  }
  .status-bar.error { background: #3a1b1b; color: #ff6b6b; }

  .grid {
    display: grid; grid-template-columns: repeat(auto-fill, minmax(140px, 1fr));
    gap: 1rem; padding: 1.5rem; overflow-y: auto; flex: 1;
  }
  .dimmed { opacity: 0.5; pointer-events: none; }

  .card { background: var(--card-bg); border-radius: 6px; overflow: hidden; }
  img { width: 100%; aspect-ratio: 2/3; object-fit: cover; display: block; }
  .name { padding: 6px; font-size: 0.75rem; color: #888; text-align: center; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }

  .empty-state {
      grid-column: 1 / -1; display: flex; justify-content: center; align-items: center;
      height: 300px; color: #555; border: 2px dashed #333; border-radius: 10px;
  }
</style>
