import { ChangeDetectionStrategy, Component, inject } from '@angular/core';
import { ReactiveFormsModule } from '@angular/forms';
import { TranslatePipe } from '@ngx-translate/core';
import { AppIcon } from '../../../../core/components/app-icon/app-icon';
import { ProfileStore } from '../../services/profile-store';

@Component({
  selector: 'app-profile-privacy',
  imports: [ReactiveFormsModule, TranslatePipe, AppIcon],
  templateUrl: './privacy.html',
  styleUrl: './privacy.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class ProfilePrivacyPage {
  protected store = inject(ProfileStore);
}
