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

import React, { useState } from 'react';
import { Typography, Card, Toast, Tooltip, Collapsible } from '@douyinfe/semi-ui';
import { Key, Globe, Copy, Shield, ChevronDown, ChevronUp } from 'lucide-react';
import CompactModeToggle from '../../common/ui/CompactModeToggle';
import { useIsMobile } from '../../../hooks/common/useIsMobile';

const { Text } = Typography;

const TokensDescription = ({ compactMode, setCompactMode, t }) => {
  const isMobile = useIsMobile();
  const [isInfoExpanded, setIsInfoExpanded] = useState(!isMobile); // 移动端默认收起，桌面端默认展开

  const copyToClipboard = (text) => {
    navigator.clipboard.writeText(text)
      .then(() => {
        Toast.success(t('接口地址已复制到剪贴板！'));
      })
      .catch(err => {
        Toast.error(t('复制失败，请重试！'));
        console.error('Failed to copy text:', err);
      });
  };

  return (
    <div className="w-full">
      {/* 主标题区域 */}
      <div className="flex flex-col md:flex-row justify-between items-start md:items-center gap-4 w-full mb-4">
        <div className="flex items-center justify-between w-full md:w-auto">
          <div className="flex items-center text-blue-500">
            <Key size={16} className="mr-2" />
            <Text>{t('KEY 管理')}</Text>
          </div>
          
          {/* 移动端显示展开/收起按钮 */}
          {isMobile && (
            <Tooltip content={isInfoExpanded ? t('收起信息') : t('展开信息')}>
              <div
                className="flex items-center justify-center w-8 h-8 rounded-lg hover:bg-gray-100 cursor-pointer transition-colors"
                onClick={() => setIsInfoExpanded(!isInfoExpanded)}
              >
                {isInfoExpanded ? 
                  <ChevronUp size={16} className="text-gray-600" /> : 
                  <ChevronDown size={16} className="text-gray-600" />
                }
              </div>
            </Tooltip>
          )}
        </div>

        <CompactModeToggle
          compactMode={compactMode}
          setCompactMode={setCompactMode}
          t={t}
        />
      </div>

      {/* 可折叠的信息卡片区域 */}
      <Collapsible isOpen={isInfoExpanded}>
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-4 mb-4">
          {/* API 地址卡片 */}
          <Card className="!rounded-xl bg-gray-50 border shadow-sm">
            <div className="flex items-center justify-between">
              <div className="flex items-center">
                <Globe size={16} className="mr-2 text-blue-600" />
                <div className="flex flex-col">
                  <Text type="secondary" className="text-xs">{t('接口地址')}</Text>
                  <Text 
                    strong 
                    className="text-gray-800 font-mono text-sm"
                  >
                    https://api.moleapi.com
                  </Text>
                </div>
              </div>
              
              <Tooltip content={t('点击复制接口地址')}>
                <div
                  className="flex items-center justify-center w-8 h-8 rounded-lg bg-white hover:bg-gray-100 cursor-pointer transition-colors border"
                  onClick={() => copyToClipboard('https://api.moleapi.com')}
                >
                  <Copy size={14} className="text-gray-600" />
                </div>
              </Tooltip>
            </div>
          </Card>

          {/* 安全提示卡片 */}
          <Card className="!rounded-xl bg-orange-50 border shadow-sm">
            <div className="flex items-center">
              <Shield size={16} className="mr-2 text-orange-600 flex-shrink-0" />
              <div>
                <Text type="secondary" className="text-xs block">{t('安全提示')}</Text>
                <Text type="tertiary" className="text-xs">
                  {t('KEY 令牌无法精确控制使用额度，请避免泄露造成损失。')}
                </Text>
              </div>
            </div>
          </Card>
        </div>
      </Collapsible>
    </div>
  );
};

export default TokensDescription; 