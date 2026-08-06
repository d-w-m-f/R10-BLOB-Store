import { Routes } from '@angular/router';
import { authGuard } from './core/guards/auth.guard';

// Usaremos lazy loading para as rotas e componentes standalone
export const routes: Routes = [
  {
    path: '', // Login page
    loadComponent: () => import('./features/auth/login/login.component').then(m => m.LoginComponent)
  },
  {
    path: 'dashboard',
    canActivate: [authGuard],
    loadComponent: () => import('./features/dashboard/dashboard-layout/dashboard-layout.component').then(m => m.DashboardLayoutComponent),
    children: [
      {
        path: '', // Redireciona /dashboard para /dashboard/files
        redirectTo: 'files',
        pathMatch: 'full'
      },
      {
        path: 'files',
        loadComponent: () => import('./features/files/file-list/file-list.component').then(m => m.FileListComponent)
      },
      {
        path: 'upload',
        loadComponent: () => import('./features/files/upload/upload.component').then(m => m.UploadComponent)
      },
      {
        path: 'management',
        loadComponent: () => import('./features/management/management-page/management-page.component').then(m => m.ManagementPageComponent)
      }
    ]
  },
  {
    path: '**', // Rota de fallback
    redirectTo: ''
  }
];
