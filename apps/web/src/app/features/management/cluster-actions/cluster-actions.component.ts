import { Component, Output, EventEmitter, OnDestroy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ManagementService, Job } from '../../../core/services/management.service';
import { Subscription, interval } from 'rxjs';
import { switchMap, filter, takeWhile } from 'rxjs/operators';

@Component({
  selector: 'app-cluster-actions',
  standalone: true,
  imports: [CommonModule],
  templateUrl: './cluster-actions.component.html',
  styleUrls: ['./cluster-actions.component.scss']
})
export class ClusterActionsComponent implements OnDestroy {
  @Output() clusterChanged = new EventEmitter<void>();

  showModal = false;
  modalAction: 'bootstrap' | 'reset' = 'bootstrap';
  
  jobStatus: Job | null = null;
  pollingSub?: Subscription;

  constructor(private mgtService: ManagementService) {}

  openModal(action: 'bootstrap' | 'reset') {
    this.modalAction = action;
    this.showModal = true;
  }

  closeModal() {
    this.showModal = false;
    this.jobStatus = null;
  }

  confirmAction() {
    if (this.modalAction === 'bootstrap') {
      this.mgtService.bootstrapCluster().subscribe(res => {
        this.startPolling(res.job_id);
      });
    } else {
      this.mgtService.resetCluster().subscribe(res => {
        this.startPolling(res.job_id);
      });
    }
  }

  startPolling(jobId: string) {
    this.pollingSub = interval(1000).pipe(
      switchMap(() => this.mgtService.getJobStatus(jobId)),
      takeWhile(job => job.job_status === 'pending' || job.job_status === 'running', true)
    ).subscribe(job => {
      this.jobStatus = job;
      if (job.job_status === 'success' || job.job_status === 'failed') {
        this.clusterChanged.emit();
        setTimeout(() => {
          if(this.jobStatus?.job_status === 'success') {
            this.closeModal();
          }
        }, 2000); 
      }
    });
  }

  ngOnDestroy() {
    this.pollingSub?.unsubscribe();
  }
}
