/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

import React from 'react';
import { Button, Typography } from '@douyinfe/semi-ui';
import { Copy, Globe, Key, ShieldAlert } from 'lucide-react';
import CompactModeToggle from '../../common/ui/CompactModeToggle';
import {
  copy,
  getServerAddress,
  showError,
  showSuccess,
} from '../../../helpers';

const { Text } = Typography;
const BASE_URL = getServerAddress();

const TokensDescription = ({ compactMode, setCompactMode, t }) => {
  const handleCopyBaseUrl = async () => {
    const copied = await copy(BASE_URL);
    if (copied) {
      showSuccess(t('已复制到剪贴板'));
    } else {
      showError(t('复制失败'));
    }
  };

  return (
    <div className='flex flex-col gap-3 w-full'>
      <div className='flex flex-col md:flex-row justify-between items-start md:items-center gap-2 w-full'>
        <div className='flex items-center text-blue-500'>
          <Key size={16} className='mr-2' />
          <Text>{t('令牌管理')}</Text>
        </div>

        <CompactModeToggle
          compactMode={compactMode}
          setCompactMode={setCompactMode}
          t={t}
        />
      </div>

      <div className='grid grid-cols-1 md:grid-cols-2 gap-3 w-full'>
        <div className='rounded-xl border border-semi-color-border bg-semi-color-bg-0 px-4 py-3 flex items-center justify-between gap-3'>
          <div className='flex items-center gap-3 min-w-0'>
            <Globe size={18} className='text-semi-color-text-2 flex-shrink-0' />
            <div className='min-w-0'>
              <Text className='!text-sm !font-medium'>{t('接口地址')}</Text>
              <div>
                <Text strong>{BASE_URL}</Text>
              </div>
            </div>
          </div>
          <Button
            icon={<Copy size={15} />}
            type='tertiary'
            theme='outline'
            onClick={handleCopyBaseUrl}
            aria-label={t('复制')}
          />
        </div>

        <div className='rounded-xl border border-semi-color-border bg-semi-color-bg-0 px-4 py-3'>
          <div className='flex items-center gap-3'>
            <ShieldAlert
              size={18}
              className='text-orange-500 flex-shrink-0'
            />
            <div className='min-w-0'>
              <Text className='!text-sm !font-medium'>{t('安全提示')}</Text>
              <div>
                <Text type='secondary'>
                  {t('KEY 令牌无法精确控制使用额度，请避免泄露造成损失。')}
                </Text>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default TokensDescription;
