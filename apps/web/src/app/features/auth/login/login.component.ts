import { Component, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormBuilder, FormGroup, ReactiveFormsModule, Validators } from '@angular/forms';
import { Router } from '@angular/router';
import { AuthService } from '../../../core/services/auth.service';

@Component({
  selector: 'app-login',
  standalone: true,
  imports: [CommonModule, ReactiveFormsModule],
  template: `
    <div class="login-wrapper">
      <div class="background-decorations">
        <div class="circle circle-1"></div>
        <div class="circle circle-2"></div>
      </div>
      
      <div class="login-container glass-panel">
        <div class="login-header">
          <i class="ph ph-hard-drives icon-logo"></i>
          <h1>R10 BLOB Store</h1>
          <p>Acesse seu armazenamento seguro</p>
        </div>

        <form [formGroup]="loginForm" (ngSubmit)="onSubmit()" class="login-form">
          <div class="input-group">
            <label for="email">E-mail</label>
            <div class="input-icon-wrapper">
              <i class="ph ph-envelope-simple icon-left"></i>
              <input 
                id="email" 
                type="email" 
                class="input-control" 
                formControlName="email" 
                placeholder="seu@email.com"
                autocomplete="email"
              />
            </div>
            <span class="error-msg" *ngIf="loginForm.get('email')?.touched && loginForm.get('email')?.invalid">
              E-mail inválido
            </span>
          </div>

          <div class="input-group">
            <label for="password">Senha</label>
            <div class="input-icon-wrapper">
              <i class="ph ph-lock-key icon-left"></i>
              <input 
                id="password" 
                type="password" 
                class="input-control" 
                formControlName="password" 
                placeholder="••••••••"
              />
            </div>
          </div>

          <button type="submit" class="btn btn-primary login-btn" [disabled]="loginForm.invalid || isLoading">
            <span *ngIf="!isLoading">Entrar</span>
            <i *ngIf="!isLoading" class="ph ph-sign-in"></i>
            <span *ngIf="isLoading" class="loader"></span>
          </button>
        </form>
      </div>
    </div>
  `,
  styleUrls: ['./login.component.scss']
})
export class LoginComponent {
  private fb = inject(FormBuilder);
  private authService = inject(AuthService);
  private router = inject(Router);

  loginForm: FormGroup = this.fb.group({
    email: ['', [Validators.required, Validators.email]],
    password: ['', [Validators.required, Validators.minLength(6)]]
  });

  isLoading = false;

  async onSubmit() {
    if (this.loginForm.valid) {
      this.isLoading = true;
      const { email, password } = this.loginForm.value;
      const success = await this.authService.login(email, password);
      this.isLoading = false;
      
      if (success) {
        this.router.navigate(['/dashboard']);
      } else {
        alert('Falha no login (mock)');
      }
    } else {
      this.loginForm.markAllAsTouched();
    }
  }
}
