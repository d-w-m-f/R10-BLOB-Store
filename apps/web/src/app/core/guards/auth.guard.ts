import { inject } from '@angular/core';
import { CanActivateFn, Router } from '@angular/router';
import { AuthService } from '../services/auth.service';

export const authGuard: CanActivateFn = (route, state) => {
  const authService = inject(AuthService);
  const router = inject(Router);

  // Lê o valor atual do signal
  if (authService.isAuthenticated()) {
    return true;
  }

  // Se não estiver logado, redireciona para a página principal (Login)
  return router.createUrlTree(['/']);
};
