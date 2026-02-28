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

import React, { useRef } from 'react';
import { Button, Dropdown } from '@douyinfe/semi-ui';
import { IconInfoCircle, IconMore } from '@douyinfe/semi-icons';
import { IconIntro } from '@douyinfe/semi-icons-lab';

const QuickNavDropdown = ({ quickNavLinks, navigate, t }) => {
  const dropdownRef = useRef(null);

  if (!quickNavLinks || quickNavLinks.length === 0) {
    return null;
  }

  const getItemIcon = (itemKey) => {
    if (itemKey === 'home') {
      return (
        <IconIntro
          size='small'
          className='text-gray-500 dark:text-gray-400'
        />
      );
    }
    if (itemKey === 'about') {
      return (
        <IconInfoCircle
          size='small'
          className='text-gray-500 dark:text-gray-400'
        />
      );
    }
    return null;
  };

  return (
    <div className='relative' ref={dropdownRef}>
      <Dropdown
        position='bottomRight'
        getPopupContainer={() => dropdownRef.current}
        render={
          <Dropdown.Menu className='!bg-semi-color-bg-overlay !border-semi-color-border !shadow-lg !rounded-lg dark:!bg-gray-700 dark:!border-gray-600'>
            {quickNavLinks.map((link) => (
              <Dropdown.Item
                key={link.itemKey}
                onClick={() => {
                  if (link.isExternal) {
                    window.open(link.externalLink, '_blank', 'noopener,noreferrer');
                  } else {
                    navigate(link.to);
                  }
                }}
                className='!px-3 !py-1.5 !text-sm !text-semi-color-text-0 hover:!bg-semi-color-fill-1 dark:!text-gray-200 dark:hover:!bg-blue-500 dark:hover:!text-white'
              >
                <div className='flex items-center gap-2'>
                  {getItemIcon(link.itemKey)}
                  <span>{link.text}</span>
                </div>
              </Dropdown.Item>
            ))}
          </Dropdown.Menu>
        }
      >
        <Button
          theme='borderless'
          type='tertiary'
          icon={<IconMore />}
          className='!h-8 !w-8 !rounded-full !bg-semi-color-fill-0 dark:!bg-semi-color-fill-1 hover:!bg-semi-color-fill-1 dark:hover:!bg-semi-color-fill-2'
          aria-label={t('更多')}
        />
      </Dropdown>
    </div>
  );
};

export default QuickNavDropdown;
