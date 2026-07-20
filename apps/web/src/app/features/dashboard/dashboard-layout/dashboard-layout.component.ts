import { Component, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { Router, RouterModule, RouterOutlet } from '@angular/router';
import { AuthService } from '../../../core/services/auth.service';

@Component({
  selector: 'app-dashboard-layout',
  standalone: true,
  imports: [CommonModule, RouterModule, RouterOutlet],
  template: `
    <div class="dashboard-wrapper">
      <!-- Sidebar navigation -->
      <aside class="sidebar glass-panel">
        <div class="sidebar-header">
          <i class="ph ph-hard-drives icon-logo"></i>
          <span class="brand">R10 Store</span>
        </div>
        
        <nav class="sidebar-nav">
          <a routerLink="/dashboard/files" routerLinkActive="active" class="nav-item">
            <i class="ph ph-folder"></i>
            <span>Meus Arquivos</span>
          </a>
          <a routerLink="/dashboard/upload" routerLinkActive="active" class="nav-item">
            <i class="ph ph-upload-simple"></i>
            <span>Upload</span>
          </a>
        </nav>

        <div class="sidebar-footer">
          <div class="user-info" *ngIf="authService.currentUser() as user">
            <i class="ph ph-user-circle"></i>
            <div class="user-details">
              <span class="user-name">{{ user.name }}</span>
              <span class="user-email">{{ user.email }}</span>
            </div>
          </div>
          <button (click)="logout()" class="btn btn-logout">
            <i class="ph ph-sign-out"></i>
            <span>Sair</span>
          </button>
        </div>
      </aside>

      <!-- Main Content Area -->
      <main class="main-content">
        <header class="topbar glass-panel">
          <div class="search-bar">
            <i class="ph ph-magnifying-glass"></i>
            <input type="text" placeholder="Buscar arquivos..." class="search-input">
          </div>
          <div class="topbar-actions">
            <button class="icon-btn"><i class="ph ph-bell"></i></button>
          </div>
        </header>
        
        <div class="content-area">
          <!-- Aqui entra o Upload ou o File List dependendo da rota ativa -->
          <router-outlet></router-outlet>
        </div>
      </main>
    </div>
  `,
  styleUrls: ['./dashboard-layout.component.scss']
})
export class DashboardLayoutComponent {
  authService = inject(AuthService);
  private router = inject(Router);

  logout() {
    this.authService.logout();
    this.router.navigate(['/']);
  }
}
