<template>
  <div class="admin-view">
    <!-- Stats Cards -->
    <section class="stats-grid">
      <div class="stat-card">
        <div class="stat-icon users">
          <font-awesome-icon icon="user" />
        </div>
        <div class="stat-info">
          <span class="stat-value">{{ stats.total_users }}</span>
          <span class="stat-label">注册用户</span>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon tasks">
          <font-awesome-icon icon="download" />
        </div>
        <div class="stat-info">
          <span class="stat-value">{{ stats.total_tasks }}</span>
          <span class="stat-label">总任务数</span>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon running">
          <font-awesome-icon icon="spinner" spin />
        </div>
        <div class="stat-info">
          <span class="stat-value">{{ stats.running_tasks }}</span>
          <span class="stat-label">运行中</span>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon completed">
          <font-awesome-icon icon="circle-check" />
        </div>
        <div class="stat-info">
          <span class="stat-value">{{ stats.completed_tasks }}</span>
          <span class="stat-label">已完成</span>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon pending">
          <font-awesome-icon icon="clock" />
        </div>
        <div class="stat-info">
          <span class="stat-value">{{ stats.pending_tasks }}</span>
          <span class="stat-label">等待中</span>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon size">
          <font-awesome-icon icon="weight-hanging" />
        </div>
        <div class="stat-info">
          <span class="stat-value">{{ formatFileSize(stats.total_files_size) }}</span>
          <span class="stat-label">文件总量</span>
        </div>
      </div>
    </section>

    <!-- Tab Switcher -->
    <div class="tab-switcher">
      <button class="tab-btn" :class="{ active: activeTab === 'users' }" @click="activeTab = 'users'">
        <font-awesome-icon icon="user" />
        <span>用户管理</span>
      </button>
      <button class="tab-btn" :class="{ active: activeTab === 'announcements' }" @click="activeTab = 'announcements'">
        <font-awesome-icon icon="bullhorn" />
        <span>公告管理</span>
      </button>
      <button class="tab-btn" :class="{ active: activeTab === 'sponsors' }" @click="activeTab = 'sponsors'">
        <font-awesome-icon icon="heart" />
        <span>赞助管理</span>
      </button>
    </div>

    <!-- Users Management -->
    <section v-if="activeTab === 'users'" class="users-section">
      <div class="section-header">
        <h2>用户管理</h2>
        <el-input
          v-model="searchQuery"
          placeholder="搜索用户（邮箱 / 用户名）"
          clearable
          class="search-input"
        />
      </div>

      <div class="table-wrapper">
        <el-table :data="filteredUsers" stripe style="width: 100%" v-loading="loading">
          <el-table-column prop="username" label="用户名" min-width="120">
            <template #default="{ row }">
              <div class="user-cell">
                <img v-if="row.avatar_url" :src="row.avatar_url" class="user-avatar" />
                <span v-else class="user-avatar-placeholder">
                  <font-awesome-icon icon="user" />
                </span>
                <span>{{ row.username }}</span>
              </div>
            </template>
          </el-table-column>
          <el-table-column prop="email" label="邮箱" min-width="160" />
          <el-table-column label="GitHub" width="80">
            <template #default="{ row }">
              <el-tooltip :content="row.github_id ? '已绑定 GitHub: ' + row.github_id : '未绑定 GitHub'">
                <font-awesome-icon
                  :icon="['fab', 'github']"
                  :style="{ color: row.github_id ? '#333' : '#ccc', fontSize: '18px' }"
                />
              </el-tooltip>
            </template>
          </el-table-column>
          <el-table-column prop="role" label="角色" width="90">
            <template #default="{ row }">
              <el-tag :type="row.role === 'admin' ? 'danger' : 'info'" size="small">
                {{ row.role === 'admin' ? '管理员' : '普通用户' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="封禁" width="80">
            <template #default="{ row }">
              <el-tag :type="row.banned ? 'danger' : 'success'" size="small">
                {{ row.banned ? '已封禁' : '正常' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="email_verified" label="邮箱验证" width="90">
            <template #default="{ row }">
              <el-tag :type="row.email_verified ? 'success' : 'warning'" size="small">
                {{ row.email_verified ? '已验证' : '未验证' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="created_at" label="注册时间" width="160">
            <template #default="{ row }">
              {{ new Date(row.created_at).toLocaleString('zh-CN') }}
            </template>
          </el-table-column>
          <el-table-column label="操作" width="220" fixed="right">
            <template #default="{ row }">
              <div class="action-btns">
                <el-button size="small" type="primary" @click="showUserDetail(row)">
                  <font-awesome-icon icon="circle-info" />
                </el-button>
                <el-popconfirm
                  v-if="!row.banned"
                  title="确定封禁此用户？"
                  confirm-button-text="确定"
                  cancel-button-text="取消"
                  @confirm="handleBan(row.id)"
                >
                  <template #reference>
                    <el-button size="small" type="warning">封禁</el-button>
                  </template>
                </el-popconfirm>
                <el-popconfirm
                  v-else
                  title="确定解封此用户？"
                  confirm-button-text="确定"
                  cancel-button-text="取消"
                  @confirm="handleUnban(row.id)"
                >
                  <template #reference>
                    <el-button size="small" type="success">解封</el-button>
                  </template>
                </el-popconfirm>
                <el-popconfirm
                  v-if="row.role === 'user'"
                  title="确定将此用户提升为管理员？"
                  confirm-button-text="确定"
                  cancel-button-text="取消"
                  @confirm="handleSetAdmin(row.id)"
                >
                  <template #reference>
                    <el-button size="small" type="info">升管</el-button>
                  </template>
                </el-popconfirm>
                <el-popconfirm
                  v-else
                  title="确定移除此用户的管理员权限？"
                  confirm-button-text="确定"
                  cancel-button-text="取消"
                  @confirm="handleRevokeAdmin(row.id)"
                >
                  <template #reference>
                    <el-button size="small" type="info">降权</el-button>
                  </template>
                </el-popconfirm>
                <el-popconfirm
                  title="确定永久删除此用户？此操作不可撤销！"
                  confirm-button-text="确定删除"
                  cancel-button-text="取消"
                  @confirm="handleDelete(row.id)"
                >
                  <template #reference>
                    <el-button size="small" type="danger">删除</el-button>
                  </template>
                </el-popconfirm>
              </div>
            </template>
          </el-table-column>
        </el-table>

      </div>

      <!-- Pagination (desktop & mobile) -->
      <div class="pagination-wrapper">
        <el-pagination
          v-model:current-page="currentPage"
          v-model:page-size="pageSize"
          :total="totalUsers"
          :page-sizes="[10, 20, 50, 100]"
          layout="total, sizes, prev, pager, next, jumper"
          background
          @current-change="handlePageChange"
          @size-change="handleSizeChange"
        />
      </div>

      <!-- Mobile card list -->
      <div class="mobile-user-cards">
        <div v-for="user in filteredUsers" :key="user.id" class="user-card">
          <div class="user-card-header">
            <div class="user-card-avatar">
              <img v-if="user.avatar_url" :src="user.avatar_url" />
              <font-awesome-icon v-else icon="user" />
            </div>
            <div class="user-card-identity">
              <div class="user-card-name-row">
                <span class="user-card-name">{{ user.username }}</span>
                <font-awesome-icon
                  v-if="user.github_id"
                  :icon="['fab', 'github']"
                  class="user-card-github"
                />
              </div>
              <span class="user-card-email">{{ user.email }}</span>
            </div>
          </div>
          <div class="user-card-tags">
            <el-tag :type="user.role === 'admin' ? 'danger' : 'info'" size="small">
              {{ user.role === 'admin' ? '管理员' : '普通用户' }}
            </el-tag>
            <el-tag :type="user.banned ? 'danger' : 'success'" size="small">
              {{ user.banned ? '已封禁' : '正常' }}
            </el-tag>
            <el-tag :type="user.email_verified ? 'success' : 'warning'" size="small">
              {{ user.email_verified ? '已验证' : '未验证' }}
            </el-tag>
          </div>
          <div class="user-card-footer">
            <span class="user-card-date">{{ new Date(user.created_at).toLocaleString('zh-CN') }}</span>
            <div class="user-card-actions">
              <el-button size="small" circle @click="showUserDetail(user)">
                <font-awesome-icon icon="circle-info" />
              </el-button>
              <el-popconfirm
                v-if="!user.banned"
                title="确定封禁此用户？"
                confirm-button-text="确定"
                cancel-button-text="取消"
                @confirm="handleBan(user.id)"
              >
                <template #reference>
                  <el-button size="small" type="warning">封禁</el-button>
                </template>
              </el-popconfirm>
              <el-popconfirm
                v-else
                title="确定解封此用户？"
                confirm-button-text="确定"
                cancel-button-text="取消"
                @confirm="handleUnban(user.id)"
              >
                <template #reference>
                  <el-button size="small" type="success">解封</el-button>
                </template>
              </el-popconfirm>
              <el-popconfirm
                title="确定永久删除？"
                confirm-button-text="删除"
                cancel-button-text="取消"
                @confirm="handleDelete(user.id)"
              >
                <template #reference>
                  <el-button size="small" type="danger">删除</el-button>
                </template>
              </el-popconfirm>
            </div>
          </div>
        </div>
      </div>

      <!-- Mobile pagination -->
      <div class="mobile-pagination-wrapper">
        <el-pagination
          v-model:current-page="currentPage"
          v-model:page-size="pageSize"
          :total="totalUsers"
          :page-sizes="[10, 20, 50, 100]"
          layout="total, prev, pager, next"
          background
          small
          @current-change="handlePageChange"
          @size-change="handleSizeChange"
        />
      </div>
    </section>

    <!-- Announcements Management -->
    <section v-if="activeTab === 'announcements'" class="users-section">
      <div class="section-header">
        <h2>公告管理</h2>
        <el-button type="primary" @click="openCreateAnnouncement">
          <font-awesome-icon icon="add" />
          <span style="margin-left:6px;">新建公告</span>
        </el-button>
      </div>

      <div class="table-wrapper">
        <el-table :data="announcements" stripe style="width: 100%" v-loading="annLoading">
          <el-table-column label="标题" min-width="160">
            <template #default="{ row }">
              <el-text tag="b" size="default">{{ row.title }}</el-text>
            </template>
          </el-table-column>
          <el-table-column label="内容" min-width="300">
            <template #default="{ row }">
              <el-popover placement="top-start" :width="400" trigger="hover" :show-after="300">
                <template #reference>
                  <el-text line-clamp="2" size="default" class="ann-content-text">
                    {{ row.content }}
                  </el-text>
                </template>
                <div class="ann-popover-content">{{ row.content }}</div>
              </el-popover>
            </template>
          </el-table-column>
          <el-table-column label="状态" width="90">
            <template #default="{ row }">
              <el-tag :type="row.is_active ? 'success' : 'info'" size="small">
                {{ row.is_active ? '启用' : '停用' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="created_at" label="创建时间" width="160">
            <template #default="{ row }">
              {{ new Date(row.created_at).toLocaleString('zh-CN') }}
            </template>
          </el-table-column>
          <el-table-column label="操作" width="200" fixed="right">
            <template #default="{ row }">
              <div class="action-btns">
                <el-button size="small" @click="toggleAnnouncement(row)">
                  {{ row.is_active ? '停用' : '启用' }}
                </el-button>
                <el-button size="small" type="primary" @click="openEditAnnouncement(row)">
                  <font-awesome-icon icon="pen" />
                </el-button>
                <el-popconfirm
                  title="确定删除此公告？"
                  confirm-button-text="确定"
                  cancel-button-text="取消"
                  @confirm="handleDeleteAnnouncement(row.id)"
                >
                  <template #reference>
                    <el-button size="small" type="danger">
                      <font-awesome-icon icon="trash" />
                    </el-button>
                  </template>
                </el-popconfirm>
              </div>
            </template>
          </el-table-column>
        </el-table>
      </div>

      <!-- Mobile announcement cards -->
      <div class="mobile-user-cards">
        <div v-for="ann in announcements" :key="ann.id" class="user-card ann-mobile-card">
          <div class="user-card-header">
            <div class="user-card-avatar" style="background: #6366f1; color: #fff;">
              <font-awesome-icon icon="bullhorn" />
            </div>
            <div class="user-card-identity">
              <el-text tag="b" size="default" class="ann-mobile-title">{{ ann.title }}</el-text>
            </div>
          </div>
          <div class="ann-mobile-content">
            <el-text line-clamp="3" size="default">{{ ann.content }}</el-text>
          </div>
          <div class="user-card-tags">
            <el-tag :type="ann.is_active ? 'success' : 'info'" size="small">
              {{ ann.is_active ? '启用' : '停用' }}
            </el-tag>
          </div>
          <div class="user-card-footer">
            <span class="user-card-date">{{ new Date(ann.created_at).toLocaleString('zh-CN') }}</span>
            <div class="user-card-actions">
              <el-button size="small" @click="toggleAnnouncement(ann)">
                {{ ann.is_active ? '停用' : '启用' }}
              </el-button>
              <el-button size="small" type="primary" @click="openEditAnnouncement(ann)">编辑</el-button>
              <el-popconfirm
                title="确定删除？"
                confirm-button-text="删除"
                cancel-button-text="取消"
                @confirm="handleDeleteAnnouncement(ann.id)"
              >
                <template #reference>
                  <el-button size="small" type="danger">删除</el-button>
                </template>
              </el-popconfirm>
            </div>
          </div>
        </div>
      </div>
    </section>

    <!-- Sponsors Management -->
    <section v-if="activeTab === 'sponsors'" class="users-section">
      <div class="section-header">
        <h2>赞助管理</h2>
        <el-button type="primary" @click="openCreateSponsor">
          <font-awesome-icon icon="add" />
          <span style="margin-left:6px;">添加赞助</span>
        </el-button>
      </div>

      <div class="table-wrapper">
        <el-table :data="sponsors" stripe style="width: 100%" v-loading="spLoading">
          <el-table-column label="名称" min-width="120">
            <template #default="{ row }">
              <div class="user-cell">
                <span class="user-avatar-placeholder">
                  <font-awesome-icon icon="user" />
                </span>
                <span>{{ row.name }}</span>
              </div>
            </template>
          </el-table-column>
          <el-table-column label="方式" width="100">
            <template #default="{ row }">
              <el-tag :type="row.method === 'wechat' ? 'success' : ''" size="small">
                {{ row.method === 'wechat' ? '微信' : '支付宝' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="amount" label="金额" width="100" />
          <el-table-column label="留言" min-width="180">
            <template #default="{ row }">
              <el-popover v-if="row.message" placement="top-start" :width="300" trigger="hover" :show-after="300">
                <template #reference>
                  <el-text line-clamp="1" size="default" class="ann-content-text">
                    {{ row.message }}
                  </el-text>
                </template>
                <div class="ann-popover-content">{{ row.message }}</div>
              </el-popover>
              <span v-else style="color: #ccc;">-</span>
            </template>
          </el-table-column>
          <el-table-column label="显示" width="80">
            <template #default="{ row }">
              <el-tag :type="row.is_visible ? 'success' : 'info'" size="small">
                {{ row.is_visible ? '可见' : '隐藏' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="created_at" label="添加时间" width="160">
            <template #default="{ row }">
              {{ new Date(row.created_at).toLocaleString('zh-CN') }}
            </template>
          </el-table-column>
          <el-table-column label="操作" width="200" fixed="right">
            <template #default="{ row }">
              <div class="action-btns">
                <el-button size="small" @click="toggleSponsor(row)">
                  {{ row.is_visible ? '隐藏' : '显示' }}
                </el-button>
                <el-button size="small" type="primary" @click="openEditSponsor(row)">
                  <font-awesome-icon icon="pen" />
                </el-button>
                <el-popconfirm
                  title="确定删除此赞助？"
                  confirm-button-text="确定"
                  cancel-button-text="取消"
                  @confirm="handleDeleteSponsor(row.id)"
                >
                  <template #reference>
                    <el-button size="small" type="danger">
                      <font-awesome-icon icon="trash" />
                    </el-button>
                  </template>
                </el-popconfirm>
              </div>
            </template>
          </el-table-column>
        </el-table>
      </div>

      <!-- Mobile sponsor cards -->
      <div class="mobile-user-cards">
        <div v-for="sp in sponsors" :key="sp.id" class="user-card">
          <div class="user-card-header">
            <div class="user-card-avatar">
              <font-awesome-icon icon="user" />
            </div>
            <div class="user-card-identity">
              <div class="user-card-name-row">
                <span class="user-card-name">{{ sp.name }}</span>
                <el-tag :type="sp.method === 'wechat' ? 'success' : ''" size="small">
                  {{ sp.method === 'wechat' ? '微信' : '支付宝' }}
                </el-tag>
              </div>
              <span v-if="sp.amount" class="user-card-email">{{ sp.amount }}</span>
            </div>
          </div>
          <p v-if="sp.message" style="margin:4px 0;font-size:13px;color:var(--text-secondary);">
            {{ sp.message }}
          </p>
          <div class="user-card-tags">
            <el-tag :type="sp.is_visible ? 'success' : 'info'" size="small">
              {{ sp.is_visible ? '可见' : '隐藏' }}
            </el-tag>
          </div>
          <div class="user-card-footer">
            <span class="user-card-date">{{ new Date(sp.created_at).toLocaleString('zh-CN') }}</span>
            <div class="user-card-actions">
              <el-button size="small" @click="toggleSponsor(sp)">
                {{ sp.is_visible ? '隐藏' : '显示' }}
              </el-button>
              <el-button size="small" type="primary" @click="openEditSponsor(sp)">编辑</el-button>
              <el-popconfirm
                title="确定删除？"
                confirm-button-text="删除"
                cancel-button-text="取消"
                @confirm="handleDeleteSponsor(sp.id)"
              >
                <template #reference>
                  <el-button size="small" type="danger">删除</el-button>
                </template>
              </el-popconfirm>
            </div>
          </div>
        </div>
      </div>
    </section>

    <!-- User Detail Dialog -->
    <el-dialog v-model="showDetailDialog" title="用户详情" width="500px" :close-on-click-modal="true">
      <template v-if="detailUser">
        <div class="user-detail">
          <div class="detail-avatar-row">
            <img v-if="detailUser.avatar_url" :src="detailUser.avatar_url" class="detail-avatar" />
            <div v-else class="detail-avatar-placeholder">
              <font-awesome-icon icon="user" size="2x" />
            </div>
            <div>
              <div class="detail-username">{{ detailUser.username }}</div>
              <div class="detail-email">{{ detailUser.email }}</div>
            </div>
          </div>
          <el-divider />
          <el-descriptions :column="1" border>
            <el-descriptions-item label="用户 ID">{{ detailUser.id }}</el-descriptions-item>
            <el-descriptions-item label="角色">
              <el-tag :type="detailUser.role === 'admin' ? 'danger' : 'info'" size="small">
                {{ detailUser.role === 'admin' ? '管理员' : '普通用户' }}
              </el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="GitHub">
              <template v-if="detailUser.github_id">
                <font-awesome-icon :icon="['fab', 'github']" style="margin-right:6px;" />
                {{ detailUser.github_id }}
              </template>
              <span v-else style="color: #ccc;">未绑定</span>
            </el-descriptions-item>
            <el-descriptions-item label="邮箱验证">
              <el-tag :type="detailUser.email_verified ? 'success' : 'warning'" size="small">
                {{ detailUser.email_verified ? '已验证' : '未验证' }}
              </el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="封禁状态">
              <el-tag :type="detailUser.banned ? 'danger' : 'success'" size="small">
                {{ detailUser.banned ? '已封禁' : '正常' }}
              </el-tag>
              <span v-if="detailUser.banned_at" style="margin-left:8px;font-size:12px;color:#999;">
                封禁时间: {{ new Date(detailUser.banned_at).toLocaleString('zh-CN') }}
              </span>
            </el-descriptions-item>
            <el-descriptions-item label="注册时间">
              {{ new Date(detailUser.created_at).toLocaleString('zh-CN') }}
            </el-descriptions-item>
            <el-descriptions-item label="最近更新">
              {{ new Date(detailUser.updated_at).toLocaleString('zh-CN') }}
            </el-descriptions-item>
          </el-descriptions>
        </div>
      </template>
      <template #footer>
        <el-button @click="showDetailDialog = false">关闭</el-button>
      </template>
    </el-dialog>

    <!-- Announcement Create/Edit Dialog -->
    <el-dialog
      v-model="showAnnDialog"
      :title="editingAnnouncement ? '编辑公告' : '新建公告'"
      width="520px"
      :close-on-click-modal="false"
    >
      <el-form :model="annForm" label-position="top">
        <el-form-item label="标题" required>
          <el-input v-model="annForm.title" placeholder="请输入公告标题" maxlength="100" show-word-limit />
        </el-form-item>
        <el-form-item label="内容" required>
          <el-input
            v-model="annForm.content"
            type="textarea"
            :rows="4"
            placeholder="请输入公告内容"
            maxlength="500"
            show-word-limit
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showAnnDialog = false" size="large" style="border-radius:20px;">取消</el-button>
        <el-button type="primary" @click="handleSaveAnnouncement" :loading="annSaving" size="large" style="border-radius:20px;">
          保存
        </el-button>
      </template>
    </el-dialog>

    <!-- Sponsor Create/Edit Dialog -->
    <el-dialog
      v-model="showSpDialog"
      :title="editingSponsor ? '编辑赞助' : '添加赞助'"
      width="520px"
      :close-on-click-modal="false"
    >
      <el-form :model="spForm" label-position="top">
        <el-form-item label="名称" required>
          <el-input v-model="spForm.name" placeholder="赞助者昵称" maxlength="50" show-word-limit />
        </el-form-item>
        <el-form-item label="方式" required>
          <el-radio-group v-model="spForm.method">
            <el-radio value="wechat">微信</el-radio>
            <el-radio value="alipay">支付宝</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="金额">
          <el-input v-model="spForm.amount" placeholder="例如：¥10 或 一杯咖啡" maxlength="50" show-word-limit />
        </el-form-item>
        <el-form-item label="留言">
          <el-input
            v-model="spForm.message"
            type="textarea"
            :rows="3"
            placeholder="赞助者的留言（选填）"
            maxlength="200"
            show-word-limit
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showSpDialog = false" size="large" style="border-radius:20px;">取消</el-button>
        <el-button type="primary" @click="handleSaveSponsor" :loading="spSaving" size="large" style="border-radius:20px;">
          保存
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import * as adminApi from '@/api/admin'
import * as announcementApi from '@/api/announcement'
import * as sponsorApi from '@/api/sponsor'
import type { User, Announcement, Sponsor } from '@/types'
import { ElMessage } from 'element-plus'

const activeTab = ref<'users' | 'announcements' | 'sponsors'>('users')

const stats = ref<adminApi.DashboardStats>({
  total_users: 0,
  total_tasks: 0,
  pending_tasks: 0,
  running_tasks: 0,
  completed_tasks: 0,
  expired_tasks: 0,
  total_files_size: 0,
})
const users = ref<User[]>([])
const loading = ref(true)
const searchQuery = ref('')

// Pagination state
const currentPage = ref(1)
const pageSize = ref(20)
const totalUsers = ref(0)

// Detail dialog
const showDetailDialog = ref(false)
const detailUser = ref<User | null>(null)

// Announcement state
const announcements = ref<Announcement[]>([])
const annLoading = ref(false)
const showAnnDialog = ref(false)
const editingAnnouncement = ref<Announcement | null>(null)
const annSaving = ref(false)
const annForm = ref({ title: '', content: '' })

// Sponsor state
const sponsors = ref<Sponsor[]>([])
const spLoading = ref(false)
const showSpDialog = ref(false)
const editingSponsor = ref<Sponsor | null>(null)
const spSaving = ref(false)
const spForm = ref({ name: '', method: 'wechat' as string, amount: '', message: '' })

const filteredUsers = computed(() => {
  if (!searchQuery.value.trim()) return users.value
  const q = searchQuery.value.toLowerCase()
  return users.value.filter(
    (u) => u.email.toLowerCase().includes(q) || u.username.toLowerCase().includes(q)
  )
})

function formatFileSize(bytes: number): string {
  if (bytes === 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(1024))
  return (bytes / Math.pow(1024, i)).toFixed(1) + ' ' + units[i]
}

function showUserDetail(user: User) {
  detailUser.value = user
  showDetailDialog.value = true
}

async function loadStats() {
  try {
    stats.value = await adminApi.getDashboard()
  } catch (e: any) {
    ElMessage.error(e.response?.data?.error || '加载统计数据失败')
  }
}

async function loadUsers(page?: number) {
  loading.value = true
  try {
    const result = await adminApi.getUsers(page || currentPage.value, pageSize.value)
    users.value = result.users
    totalUsers.value = result.total
    currentPage.value = result.page
  } catch (e: any) {
    ElMessage.error(e.response?.data?.error || '加载用户数据失败')
  } finally {
    loading.value = false
  }
}

function handlePageChange(page: number) {
  currentPage.value = page
  loadUsers(page)
}

function handleSizeChange(size: number) {
  pageSize.value = size
  currentPage.value = 1
  loadUsers(1)
}

async function loadData() {
  await Promise.all([loadStats(), loadUsers()])
}

async function handleSetAdmin(userId: string) {
  try {
    await adminApi.updateUserRole(userId, 'admin')
    ElMessage.success('已设置为管理员')
    await loadUsers()
  } catch (e: any) {
    ElMessage.error(e.response?.data?.error || '操作失败')
  }
}

async function handleRevokeAdmin(userId: string) {
  try {
    await adminApi.updateUserRole(userId, 'user')
    ElMessage.success('已取消管理员权限')
    await loadUsers()
  } catch (e: any) {
    ElMessage.error(e.response?.data?.error || '操作失败')
  }
}

async function handleBan(userId: string) {
  try {
    await adminApi.banUser(userId)
    ElMessage.success('用户已封禁')
    await loadUsers()
  } catch (e: any) {
    ElMessage.error(e.response?.data?.error || '封禁失败')
  }
}

async function handleUnban(userId: string) {
  try {
    await adminApi.unbanUser(userId)
    ElMessage.success('用户已解封')
    await loadUsers()
  } catch (e: any) {
    ElMessage.error(e.response?.data?.error || '解封失败')
  }
}

async function handleDelete(userId: string) {
  try {
    await adminApi.deleteUser(userId)
    ElMessage.success('用户已删除')
    await loadUsers()
  } catch (e: any) {
    ElMessage.error(e.response?.data?.error || '删除失败')
  }
}

// --- Announcement methods ---
async function loadAnnouncements() {
  annLoading.value = true
  try {
    announcements.value = await announcementApi.getAllAnnouncements()
  } catch (e: any) {
    ElMessage.error(e.response?.data?.error || '加载公告失败')
  } finally {
    annLoading.value = false
  }
}

function openCreateAnnouncement() {
  editingAnnouncement.value = null
  annForm.value = { title: '', content: '' }
  showAnnDialog.value = true
}

function openEditAnnouncement(ann: Announcement) {
  editingAnnouncement.value = ann
  annForm.value = { title: ann.title, content: ann.content }
  showAnnDialog.value = true
}

async function handleSaveAnnouncement() {
  const title = annForm.value.title.trim()
  const content = annForm.value.content.trim()
  if (!title) { ElMessage.warning('标题不能为空'); return }
  if (!content) { ElMessage.warning('内容不能为空'); return }

  annSaving.value = true
  try {
    if (editingAnnouncement.value) {
      await announcementApi.updateAnnouncement(editingAnnouncement.value.id, { title, content })
      ElMessage.success('公告已更新')
    } else {
      await announcementApi.createAnnouncement({ title, content })
      ElMessage.success('公告已创建')
    }
    showAnnDialog.value = false
    await loadAnnouncements()
  } catch (e: any) {
    ElMessage.error(e.response?.data?.error || '保存失败')
  } finally {
    annSaving.value = false
  }
}

async function toggleAnnouncement(ann: Announcement) {
  try {
    await announcementApi.updateAnnouncement(ann.id, {
      title: ann.title,
      content: ann.content,
      is_active: !ann.is_active,
    })
    ElMessage.success(ann.is_active ? '公告已停用' : '公告已启用')
    await loadAnnouncements()
  } catch (e: any) {
    ElMessage.error(e.response?.data?.error || '操作失败')
  }
}

async function handleDeleteAnnouncement(id: string) {
  try {
    await announcementApi.deleteAnnouncement(id)
    ElMessage.success('公告已删除')
    await loadAnnouncements()
  } catch (e: any) {
    ElMessage.error(e.response?.data?.error || '删除失败')
  }
}

// --- Sponsor methods ---
async function loadSponsors() {
  spLoading.value = true
  try {
    sponsors.value = await sponsorApi.getAllSponsors()
  } catch (e: any) {
    ElMessage.error(e.response?.data?.error || '加载赞助列表失败')
  } finally {
    spLoading.value = false
  }
}

function openCreateSponsor() {
  editingSponsor.value = null
  spForm.value = { name: '', method: 'wechat', amount: '', message: '' }
  showSpDialog.value = true
}

function openEditSponsor(sp: Sponsor) {
  editingSponsor.value = sp
  spForm.value = { name: sp.name, method: sp.method, amount: sp.amount, message: sp.message }
  showSpDialog.value = true
}

async function handleSaveSponsor() {
  const name = spForm.value.name.trim()
  const method = spForm.value.method.trim()
  if (!name) { ElMessage.warning('名称不能为空'); return }
  if (method !== 'wechat' && method !== 'alipay') { ElMessage.warning('请选择支付方式'); return }

  spSaving.value = true
  try {
    if (editingSponsor.value) {
      await sponsorApi.updateSponsor(editingSponsor.value.id, {
        name,
        method,
        amount: spForm.value.amount.trim(),
        message: spForm.value.message.trim(),
      })
      ElMessage.success('赞助已更新')
    } else {
      await sponsorApi.createSponsor({
        name,
        method,
        amount: spForm.value.amount.trim(),
        message: spForm.value.message.trim(),
      })
      ElMessage.success('赞助已添加')
    }
    showSpDialog.value = false
    await loadSponsors()
  } catch (e: any) {
    ElMessage.error(e.response?.data?.error || '保存失败')
  } finally {
    spSaving.value = false
  }
}

async function toggleSponsor(sp: Sponsor) {
  try {
    await sponsorApi.updateSponsor(sp.id, {
      name: sp.name,
      method: sp.method,
      amount: sp.amount,
      message: sp.message,
      is_visible: !sp.is_visible,
    })
    ElMessage.success(sp.is_visible ? '已隐藏' : '已显示')
    await loadSponsors()
  } catch (e: any) {
    ElMessage.error(e.response?.data?.error || '操作失败')
  }
}

async function handleDeleteSponsor(id: string) {
  try {
    await sponsorApi.deleteSponsor(id)
    ElMessage.success('赞助已删除')
    await loadSponsors()
  } catch (e: any) {
    ElMessage.error(e.response?.data?.error || '删除失败')
  }
}

onMounted(() => {
  loadData()
  loadAnnouncements()
  loadSponsors()
})
</script>

<style scoped>
.admin-view {
  max-width: 1200px;
  margin: 0 auto;
  padding: 0 16px;
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
  gap: 16px;
  margin-bottom: 32px;
}

.stat-card {
  background: var(--bg-card);
  border-radius: 12px;
  padding: 20px;
  display: flex;
  align-items: center;
  gap: 16px;
  box-shadow: var(--shadow-sm);
}

.stat-icon {
  width: 48px;
  height: 48px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 20px;
  color: #fff;
}

.stat-icon.users { background: #6366f1; }
.stat-icon.tasks { background: #3b82f6; }
.stat-icon.running { background: #f59e0b; }
.stat-icon.completed { background: #10b981; }
.stat-icon.pending { background: #8b5cf6; }
.stat-icon.size { background: #06b6d4; }

.stat-info {
  display: flex;
  flex-direction: column;
}

.stat-value {
  font-size: 24px;
  font-weight: 700;
  color: var(--text-primary);
  line-height: 1.2;
}

.stat-label {
  font-size: 13px;
  color: var(--text-secondary);
  margin-top: 2px;
}

.users-section {
  background: var(--bg-card);
  border-radius: 12px;
  padding: 24px;
  box-shadow: var(--shadow-sm);
}

.section-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 20px;
  gap: 16px;
}

.section-header h2 {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
  white-space: nowrap;
}

.search-input {
  max-width: 300px;
}

/* Tab switcher */
.tab-switcher {
  display: flex;
  gap: 8px;
  margin-bottom: 20px;
}

.tab-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 10px 20px;
  border: 1px solid var(--border-color, #e0e0e0);
  border-radius: 10px;
  background: var(--bg-card);
  color: var(--text-secondary);
  font-size: 14px;
  cursor: pointer;
  transition: all 0.2s;
}

.tab-btn:hover {
  border-color: #6366f1;
  color: #6366f1;
}

.tab-btn.active {
  background: #6366f1;
  color: #fff;
  border-color: #6366f1;
}

/* Announcement table content */
.ann-content-text {
  cursor: pointer;
  color: var(--text-secondary);
  line-height: 1.6;
}

.ann-popover-content {
  white-space: pre-wrap;
  word-break: break-word;
  line-height: 1.6;
  font-size: 14px;
  color: var(--text-primary);
}

/* Mobile announcement cards */
.ann-mobile-card .user-card-header {
  align-items: flex-start;
}

.ann-mobile-title {
  flex: 1;
  line-height: 1.5;
  word-break: break-word;
}

.ann-mobile-content {
  padding: 0;
  line-height: 1.6;
  color: var(--text-secondary);
  font-size: 14px;
}

.user-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}

.user-avatar {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  object-fit: cover;
}

.user-avatar-placeholder {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  background: var(--bg-secondary);
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-secondary);
  font-size: 14px;
}

.action-btns {
  display: flex;
  gap: 4px;
  flex-wrap: wrap;
}

/* User detail dialog */
.user-detail {
  padding: 0;
}

.detail-avatar-row {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-bottom: 8px;
}

.detail-avatar {
  width: 64px;
  height: 64px;
  border-radius: 50%;
  object-fit: cover;
}

.detail-avatar-placeholder {
  width: 64px;
  height: 64px;
  border-radius: 50%;
  background: var(--bg-secondary, #f0f0f0);
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-secondary, #999);
}

.detail-username {
  font-size: 18px;
  font-weight: 600;
  color: var(--text-primary);
}

.detail-email {
  font-size: 13px;
  color: var(--text-secondary);
  margin-top: 2px;
}

/* ======== Mobile user cards (hidden on desktop) ======== */
.mobile-user-cards {
  display: none;
}

.table-wrapper {
  overflow-x: auto;
  -webkit-overflow-scrolling: touch;
}

.pagination-wrapper {
  display: flex;
  justify-content: center;
  margin-top: 20px;
  padding-top: 16px;
  border-top: 1px solid var(--border-color, #e0e0e0);
}

.mobile-pagination-wrapper {
  display: none;
  justify-content: center;
  margin-top: 16px;
  padding-top: 12px;
  border-top: 1px solid var(--border-color, #e0e0e0);
}

/* ======== Responsive: Mobile ======== */
@media (max-width: 768px) {
  .admin-view {
    padding: 0 8px;
  }

  /* Tab switcher mobile */
  .tab-switcher {
    gap: 6px;
    margin-bottom: 16px;
  }

  .tab-btn {
    flex: 1;
    justify-content: center;
    padding: 8px 12px;
    font-size: 13px;
  }

  /* Stats: 2 columns on mobile */
  .stats-grid {
    grid-template-columns: repeat(2, 1fr);
    gap: 10px;
    margin-bottom: 20px;
  }

  .stat-card {
    padding: 14px 12px;
    gap: 10px;
    border-radius: 10px;
  }

  .stat-icon {
    width: 38px;
    height: 38px;
    border-radius: 10px;
    font-size: 16px;
  }

  .stat-value {
    font-size: 20px;
  }

  .stat-label {
    font-size: 11px;
  }

  /* Users section */
  .users-section {
    padding: 16px 12px;
    border-radius: 10px;
  }

  .section-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 12px;
    margin-bottom: 16px;
  }

  .section-header h2 {
    font-size: 16px;
  }

  .search-input {
    max-width: 100%;
    width: 100%;
  }

  /* Dialog responsive: full-width on mobile */
  :deep(.el-dialog) {
    width: 92% !important;
    margin: 0 auto;
  }

  :deep(.el-dialog__body) {
    padding: 16px 12px;
  }

  :deep(.el-descriptions) {
    font-size: 12px;
  }

  /* Hide table and desktop pagination, show cards on mobile */
  .table-wrapper {
    display: none;
  }

  .pagination-wrapper {
    display: none;
  }

  .mobile-pagination-wrapper {
    display: flex;
  }

  .mobile-user-cards {
    display: flex;
    flex-direction: column;
    gap: 10px;
  }

  .user-card {
    background: #fff;
    border-radius: 10px;
    box-shadow: 0 1px 3px rgba(0,0,0,0.08);
    padding: 14px;
    display: flex;
    flex-direction: column;
    gap: 10px;
  }

  .user-card-header {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .user-card-avatar {
    width: 40px;
    height: 40px;
    border-radius: 50%;
    overflow: hidden;
    flex-shrink: 0;
    background: var(--bg-secondary, #f0f0f0);
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--text-secondary, #999);
    font-size: 18px;
  }

  .user-card-avatar img {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }

  .user-card-identity {
    flex: 1;
    min-width: 0;
  }

  .user-card-name-row {
    display: flex;
    align-items: center;
    gap: 6px;
  }

  .user-card-name {
    font-size: 14px;
    font-weight: 600;
    color: var(--text-primary, #1a1a1a);
    line-height: 1.3;
  }

  .user-card-github {
    font-size: 15px;
    color: #333;
    flex-shrink: 0;
  }

  .user-card-email {
    display: block;
    font-size: 12px;
    color: var(--text-secondary, #999);
    margin-top: 2px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .user-card-tags {
    display: flex;
    gap: 6px;
    flex-wrap: wrap;
  }

  .user-card-footer {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 8px;
    flex-wrap: wrap;
  }

  .user-card-date {
    font-size: 11px;
    color: var(--text-secondary, #999);
  }

  .user-card-actions {
    display: flex;
    gap: 4px;
  }
}
</style>
