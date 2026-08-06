import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';
import { environment } from '../../../environments/environment';

export interface ClusterStats {
  workers: number;
  machines: number;
  capacity_mb: number;
  used_mb: number;
}

export interface Worker {
  worker_uuid: string;
  worker_name: string;
  worker_capacity_mb: number;
  worker_used_mb: number;
  worker_status: string;
  machines?: Machine[];
}

export interface Machine {
  machine_uuid: string;
  machine_name: string;
  machine_type: string;
  machine_worker_id: string;
}

export interface Job {
  job_uuid: string;
  job_type: string;
  job_status: string;
  job_error?: string;
}

export interface JobResponse {
  job_id: string;
  message: string;
}

@Injectable({
  providedIn: 'root'
})
export class ManagementService {
  private apiUrl = environment.managementApiUrl;

  constructor(private http: HttpClient) {}

  getClusterStats(): Observable<ClusterStats> {
    return this.http.get<ClusterStats>(`${this.apiUrl}/cluster`);
  }

  getWorkers(): Observable<Worker[]> {
    return this.http.get<Worker[]>(`${this.apiUrl}/workers`);
  }

  bootstrapCluster(): Observable<JobResponse> {
    return this.http.post<JobResponse>(`${this.apiUrl}/bootstrap`, {});
  }

  resetCluster(): Observable<JobResponse> {
    return this.http.post<JobResponse>(`${this.apiUrl}/reset`, {});
  }

  getJobStatus(jobId: string): Observable<Job> {
    return this.http.get<Job>(`${this.apiUrl}/jobs/${jobId}`);
  }
}
