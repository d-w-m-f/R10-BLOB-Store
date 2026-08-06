import { Component, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ClusterActionsComponent } from '../cluster-actions/cluster-actions.component';
import { MonitoringDashboardComponent } from '../monitoring-dashboard/monitoring-dashboard.component';
import { DaemonVisibilityComponent } from '../daemon-visibility/daemon-visibility.component';

@Component({
  selector: 'app-management-page',
  standalone: true,
  imports: [CommonModule, ClusterActionsComponent, MonitoringDashboardComponent, DaemonVisibilityComponent],
  templateUrl: './management-page.component.html',
  styleUrls: ['./management-page.component.scss']
})
export class ManagementPageComponent implements OnInit {
  refreshTrigger = 0;

  constructor() {}

  ngOnInit(): void {}

  onClusterChanged(): void {
    this.refreshTrigger++;
  }
}
