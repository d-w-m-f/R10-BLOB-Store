import { Component, signal } from '@angular/core';
import { CommonModule } from '@angular/common';

@Component({
  selector: 'app-upload',
  standalone: true,
  imports: [CommonModule],
  template: `
    <div class="page-container fade-in">
      <header class="page-header">
        <h2>Upload de Arquivos</h2>
        <p>Arraste e solte seus arquivos para enviar ao R10 BLOB Store.</p>
      </header>

      <div class="upload-area glass-panel" 
           [class.drag-over]="isDragging()"
           (dragover)="onDragOver($event)"
           (dragleave)="onDragLeave($event)"
           (drop)="onDrop($event)">
        
        <div class="upload-content">
          <i class="ph ph-cloud-arrow-up icon-huge"></i>
          <h3>Arraste seus arquivos aqui</h3>
          <p>ou</p>
          <button class="btn btn-primary" (click)="fileInput.click()">
            <i class="ph ph-folder-open"></i> Explorar arquivos
          </button>
          <input type="file" #fileInput multiple hidden (change)="onFileSelect($event)" />
        </div>
      </div>

      <div class="upload-progress-list" *ngIf="uploadingFiles().length > 0">
        <h3>Enviando...</h3>
        <div class="file-item glass-panel" *ngFor="let file of uploadingFiles()">
          <div class="file-info">
            <i class="ph ph-file"></i>
            <span class="file-name">{{ file.name }}</span>
          </div>
          <div class="progress-bar-container">
            <div class="progress-bar" [style.width.%]="file.progress"></div>
          </div>
          <span class="progress-text">{{ file.progress }}%</span>
        </div>
      </div>
    </div>
  `,
  styleUrls: ['./upload.component.scss']
})
export class UploadComponent {
  isDragging = signal<boolean>(false);
  uploadingFiles = signal<{name: string, progress: number}[]>([]);

  onDragOver(event: DragEvent) {
    event.preventDefault();
    this.isDragging.set(true);
  }

  onDragLeave(event: DragEvent) {
    event.preventDefault();
    this.isDragging.set(false);
  }

  onDrop(event: DragEvent) {
    event.preventDefault();
    this.isDragging.set(false);
    
    if (event.dataTransfer?.files) {
      this.handleFiles(Array.from(event.dataTransfer.files));
    }
  }

  onFileSelect(event: Event) {
    const input = event.target as HTMLInputElement;
    if (input.files) {
      this.handleFiles(Array.from(input.files));
    }
  }

  private handleFiles(files: File[]) {
    // Adiciona os arquivos à lista com progresso 0
    const newFiles = files.map(f => ({ name: f.name, progress: 0 }));
    this.uploadingFiles.update(current => [...current, ...newFiles]);

    // Simula o upload progredindo
    newFiles.forEach((file, index) => {
      let progress = 0;
      const interval = setInterval(() => {
        progress += Math.floor(Math.random() * 20) + 10;
        if (progress >= 100) {
          progress = 100;
          clearInterval(interval);
          // Opcional: remover da lista ao terminar e adicionar na store real
        }
        
        this.uploadingFiles.update(current => {
          const updated = [...current];
          const fileToUpdate = updated.find(f => f.name === file.name);
          if (fileToUpdate) fileToUpdate.progress = progress;
          return updated;
        });
      }, 500 + Math.random() * 500); // tempo aleatório para cada arquivo
    });
  }
}
