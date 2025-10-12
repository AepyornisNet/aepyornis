import { Component, inject, computed } from '@angular/core';
import { AppConfig } from '../../../core/services/app-config';

@Component({
  selector: 'app-footer',
  imports: [],
  templateUrl: './footer.html',
  styleUrl: './footer.scss'
})
export class Footer {
  private appConfig = inject(AppConfig);

  version = computed(() => this.appConfig.getVersion());
  versionSha = computed(() => this.appConfig.getVersionSha());
}
