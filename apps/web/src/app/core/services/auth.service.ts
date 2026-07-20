import { Injectable, signal } from '@angular/core';

@Injectable({
  providedIn: 'root'
})
export class AuthService {
  // O signal() é uma feature nova do Angular para reatividade moderna. 
  // Ele avisa a interface sempre que o valor mudar, de forma muito mais simples que o RxJS puro.
  isAuthenticated = signal<boolean>(false);
  currentUser = signal<{name: string, email: string} | null>(null);

  constructor() {
    // Ao iniciar o serviço, verifica se já existe sessão mockada
    const token = localStorage.getItem('r10_auth_token');
    if (token) {
      this.isAuthenticated.set(true);
      this.currentUser.set({ name: 'Admin User', email: 'admin@r10.com' });
    }
  }

  // Simula um login assíncrono com Promise (na vida real usaríamos HttpClient com Observables)
  async login(email: string, pass: string): Promise<boolean> {
    return new Promise(resolve => {
      setTimeout(() => {
        if (email && pass) {
          localStorage.setItem('r10_auth_token', 'mock_token_12345');
          this.isAuthenticated.set(true);
          this.currentUser.set({ name: 'Admin User', email });
          resolve(true);
        } else {
          resolve(false);
        }
      }, 1000); // 1 segundo de mock
    });
  }

  logout() {
    localStorage.removeItem('r10_auth_token');
    this.isAuthenticated.set(false);
    this.currentUser.set(null);
  }
}
