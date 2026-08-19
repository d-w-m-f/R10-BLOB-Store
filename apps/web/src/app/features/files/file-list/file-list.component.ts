import { Component, OnInit, signal, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FileService, BlobFile } from '../../../core/services/file.service';
import { ManagementService, ClusterStats } from '../../../core/services/management.service';

@Component({
  selector: 'app-file-list',
  standalone: true,
  imports: [CommonModule],
  template: `
    <div class="page-container fade-in">
      <header class="page-header flex-between">
        <div>
          <h2>Meus Arquivos</h2>
          <p>Gerencie seus arquivos armazenados no R10 Blob Store.</p>
        </div>
        <div>
          <button class="btn btn-secondary" (click)="refresh()" title="Atualizar dados">
            <i class="ph ph-arrows-clockwise"></i> Atualizar
          </button>
        </div>
      </header>

      <!-- Storage Quota Widget (Real Data from Cluster Management API) -->
      <section class="storage-widget glass-panel">
        <div class="storage-info">
          <div class="storage-text">
            <span class="used-amount">{{ usedFormatted() }}</span> de <span class="total-amount">{{ totalFormatted() }}</span> utilizados
          </div>
          <div class="storage-percentage">{{ percentage() }}%</div>
        </div>
        <div class="storage-bar-bg">
          <div class="storage-bar-fill" [style.width.%]="percentage()"></div>
        </div>
      </section>

      <!-- Files Table (Live Data from Gateway Catalog) -->
      <section class="files-table-container glass-panel">
        <table class="files-table">
          <thead>
            <tr>
              <th>Nome do Arquivo</th>
              <th>Tamanho</th>
              <th>Data de Envio</th>
              <th class="actions-col">Ações</th>
            </tr>
          </thead>
          <tbody>
            <tr *ngFor="let file of files()">
              <td>
                <div class="file-name-cell">
                  <i class="ph ph-file-{{ getFileIcon(file) }}"></i>
                  <span>{{ file.blob_filename }}</span>
                </div>
              </td>
              <td>{{ formatSize(file.blob_size) }}</td>
              <td>{{ formatDate(file.blob_created_at) }}</td>
              <td class="actions-col">
                <button class="btn-icon" title="Baixar arquivo" (click)="downloadFile(file)">
                  <i class="ph ph-download-simple"></i>
                </button>
                <button class="btn-icon danger" title="Deletar arquivo" (click)="deleteFile(file)">
                  <i class="ph ph-trash"></i>
                </button>
              </td>
            </tr>
            <!-- Empty state if no files -->
            <tr *ngIf="files().length === 0">
              <td colspan="4" class="empty-state">
                Nenhum arquivo encontrado. Faça upload para começar.
              </td>
            </tr>
          </tbody>
        </table>
      </section>
    </div>
  `,
  styleUrls: ['./file-list.component.scss']
})
export class FileListComponent implements OnInit {
  private fileService = inject(FileService);
  private mgtService = inject(ManagementService);

  files = signal<BlobFile[]>([]);
  usedFormatted = signal<string>('0 MB');
  totalFormatted = signal<string>('0 MB');
  percentage = signal<number>(0);

  ngOnInit() {
    this.refresh();
  }

  refresh() {
    this.loadStats();
    this.loadFiles();
  }

  loadStats() {
    this.mgtService.getClusterStats().subscribe({
      next: (stats: ClusterStats) => {
        const cap = stats.capacity_mb || 0;
        const used = stats.used_mb || 0;
        
        this.totalFormatted.set(this.formatMb(cap));
        this.usedFormatted.set(this.formatMb(used));
        
        if (cap > 0) {
          const pct = Math.min(100, Math.round((used / cap) * 100));
          this.percentage.set(pct);
        } else {
          this.percentage.set(0);
        }
      },
      error: (err) => {
        console.warn('Não foi possível obter dados do cluster R10 (Pode estar desconectado):', err);
        this.totalFormatted.set('0 MB');
        this.usedFormatted.set('0 MB');
        this.percentage.set(0);
      }
    });
  }

  loadFiles() {
    this.fileService.listFiles().subscribe({
      next: (data: BlobFile[]) => {
        this.files.set(data || []);
      },
      error: (err) => {
        console.error('Erro ao listar arquivos do Gateway:', err);
        this.files.set([]);
      }
    });
  }

  formatMb(mb: number): string {
    if (mb === 0) return '0 MB';
    if (mb >= 1024) {
      return (mb / 1024).toFixed(1) + ' GB';
    }
    return mb + ' MB';
  }

  formatSize(bytes: number): string {
    return this.fileService.formatBytes(bytes);
  }

  formatDate(dateStr: string): string {
    if (!dateStr) return '-';
    try {
      const d = new Date(dateStr);
      return d.toLocaleDateString('pt-BR', { day: '2-digit', month: 'short', year: 'numeric' });
    } catch {
      return dateStr;
    }
  }

  getFileIcon(file: BlobFile): string {
    return this.fileService.getFileIcon(file.blob_filename, file.blob_mime_type);
  }

  downloadFile(file: BlobFile) {
    this.fileService.downloadFile(file.blob_uuid).subscribe({
      next: (payload) => {
        // Hand the reassembled bytes to the browser via a temporary object URL.
        const url = URL.createObjectURL(payload);
        const link = document.createElement('a');
        link.href = url;
        link.download = file.blob_filename;
        link.click();
        URL.revokeObjectURL(url);
      },
      error: (err) => {
        alert(`Falha ao baixar "${file.blob_filename}": ${err?.error?.error ?? err.message}`);
      }
    });
  }

  deleteFile(file: BlobFile) {
    if (confirm(`Tem certeza que deseja deletar o arquivo "${file.blob_filename}"?`)) {
      this.fileService.deleteFile(file.blob_uuid).subscribe({
        next: () => {
          this.refresh();
        },
        error: (err) => {
          console.error('Erro ao deletar arquivo:', err);
          alert('Não foi possível deletar o arquivo.');
        }
      });
    }
  }
}
