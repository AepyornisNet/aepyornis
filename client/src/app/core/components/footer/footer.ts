import { Component, computed, inject } from '@angular/core';
import { AppConfig } from '../../../core/services/app-config';
import { TranslatePipe } from '@ngx-translate/core';

@Component({
  selector: 'app-footer',
  imports: [TranslatePipe],
  templateUrl: './footer.html',
  styleUrl: './footer.scss',
})
export class Footer {
  private appConfig = inject(AppConfig);

  readonly version = computed(() => this.appConfig.getVersion());
  readonly versionSha = computed(() => this.appConfig.getVersionSha());
}
