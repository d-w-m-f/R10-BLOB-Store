import { Component, signal } from '@angular/core';
import { CommonModule } from '@angular/common';

interface AppFile {
  id: string;
  name: string;
  size: string;
  date: string;
  type: string;
}

@Component({
  selector: 'app-file-list',
  standalone: true,
  imports: [CommonModule],
  template: `
    <div class="page-container fade-in">
      <header class="page-header flex-between">
        <div>
          <h2>Meus Arquivos</h2>
          <p>Gerencie seus arquivos armazenados no R10.</p>
        </div>
      </header>

      <!-- Storage Quota Widget -->
      <section class="storage-widget glass-panel">
        <div class="storage-info">
          <div class="storage-text">
            <span class="used-amount">45 GB</span> de <span class="total-amount">100 GB</span> utilizados
          </div>
          <div class="storage-percentage">45%</div>
        </div>
        <div class="storage-bar-bg">
          <div class="storage-bar-fill" style="width: 45%;"></div>
        </div>
      </section>

      <!-- Files Table -->
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
                  <i class="ph ph-file-{{ getFileIcon(file.type) }}"></i>
                  <span>{{ file.name }}</span>
                </div>
              </td>
              <td>{{ file.size }}</td>
              <td>{{ file.date }}</td>
              <td class="actions-col">
                <button class="btn-icon" title="Baixar arquivo">
                  <i class="ph ph-download-simple"></i>
                </button>
                <button class="btn-icon danger" title="Deletar arquivo">
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
export class FileListComponent {
  files = signal<AppFile[]>([
    { id: '1', name: 'relatorio_anual_2026.pdf', size: '2.4 MB', date: '20 Jul 2026', type: 'pdf' },
    { id: '2', name: 'design_system.zip', size: '45.1 MB', date: '18 Jul 2026', type: 'zip' },
    { id: '3', name: 'foto_perfil.jpg', size: '1.2 MB', date: '15 Jul 2026', type: 'image' },
    { id: '4', name: 'planilha_custos.xlsx', size: '540 KB', date: '10 Jul 2026', type: 'excel' },
  ]);

  getFileIcon(type: string): string {
    switch (type) {
      case 'pdf': return 'pdf';
      case 'zip': return 'zip';
      case 'image': return 'image';
      case 'excel': return 'xls';
      default: return 'text';
    }
  }
}
