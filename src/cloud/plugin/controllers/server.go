/*
 * Copyright 2018- The Pixie Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package controllers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/gofrs/uuid"
	"github.com/gogo/protobuf/types"
	"github.com/jmoiron/sqlx"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"gopkg.in/yaml.v2"

	"px.dev/pixie/src/api/proto/uuidpb"
	"px.dev/pixie/src/cloud/cron_script/cronscriptpb"
	"px.dev/pixie/src/cloud/plugin/pluginpb"
	"px.dev/pixie/src/shared/scripts"
	"px.dev/pixie/src/shared/services/authcontext"
	"px.dev/pixie/src/utils"
)

// Server is a bridge implementation of the pluginService.
type Server struct {
	db    *sqlx.DB
	dbKey string

	cronScriptClient cronscriptpb.CronScriptServiceClient

	done chan struct{}
	once sync.Once
}

// New creates a new server.
func New(db *sqlx.DB, dbKey string, cronScriptClient cronscriptpb.CronScriptServiceClient) *Server {
	return &Server{
		db:               db,
		dbKey:            dbKey,
		cronScriptClient: cronScriptClient,
		done:             make(chan struct{}),
	}
}

// Stop performs any necessary cleanup before shutdown.
func (s *Server) Stop() {
	s.once.Do(func() {
		close(s.done)
	})
}

func contextWithAuthToken(ctx context.Context) (context.Context, error) {
	sCtx, err := authcontext.FromContext(ctx)
	if err != nil {
		return nil, err
	}
	return metadata.AppendToOutgoingContext(ctx, "authorization",
		fmt.Sprintf("bearer %s", sCtx.AuthToken)), nil
}

// PluginService implementation.

// Plugin contains metadata about a plugin.
type Plugin struct {
	Name                 string  `db:"name"`
	ID                   string  `db:"id"`
	Description          *string `db:"description"`
	Logo                 *string `db:"logo"`
	Version              string  `db:"version"`
	DataRetentionEnabled bool    `db:"data_retention_enabled" yaml:"dataRetentionEnabled"`
}

// GetPlugins fetches all of the available, latest plugins.
func (s *Server) GetPlugins(ctx context.Context, req *pluginpb.GetPluginsRequest) (*pluginpb.GetPluginsResponse, error) {
	query := `SELECT t1.name, t1.id, t1.description, t1.logo, t1.version, t1.data_retention_enabled FROM plugin_releases t1
		JOIN (SELECT id, MAX(version) as version FROM plugin_releases GROUP BY id) t2
	  	ON t1.id = t2.id AND t1.version = t2.version`

	if req.Kind == pluginpb.PLUGIN_KIND_RETENTION {
		query = fmt.Sprintf("%s %s", query, "WHERE data_retention_enabled='true'")
	}

	rows, err := s.db.Queryx(query)
	if err != nil {
		if err == sql.ErrNoRows {
			return &pluginpb.GetPluginsResponse{Plugins: nil}, nil
		}
		return nil, status.Errorf(codes.Internal, "Failed to fetch plugins")
	}
	defer rows.Close()

	plugins := []*pluginpb.Plugin{}
	for rows.Next() {
		var p Plugin
		err = rows.StructScan(&p)
		if err != nil {
			return nil, status.Error(codes.Internal, "failed to read plugins")
		}
		ppb := &pluginpb.Plugin{
			Name:             p.Name,
			ID:               p.ID,
			LatestVersion:    p.Version,
			RetentionEnabled: p.DataRetentionEnabled,
		}
		if p.Description != nil {
			ppb.Description = *p.Description
		}
		if p.Logo != nil {
			ppb.Logo = *p.Logo
		}
		plugins = append(plugins, ppb)
	}
	return &pluginpb.GetPluginsResponse{Plugins: plugins}, nil
}

// RetentionPlugin contains metadata about a retention plugin.
type RetentionPlugin struct {
	ID                   string         `db:"plugin_id"`
	Version              string         `db:"version"`
	Configurations       Configurations `db:"configurations"`
	DocumentationURL     *string        `db:"documentation_url" yaml:"documentationURL"`
	DefaultExportURL     *string        `db:"default_export_url" yaml:"defaultExportURL"`
	AllowCustomExportURL bool           `db:"allow_custom_export_url" yaml:"allowCustomExportURL"`
	PresetScripts        PresetScripts  `db:"preset_scripts" yaml:"presetScripts"`
}

// GetRetentionPluginConfig gets the config for a specific plugin release.
func (s *Server) GetRetentionPluginConfig(ctx context.Context, req *pluginpb.GetRetentionPluginConfigRequest) (*pluginpb.GetRetentionPluginConfigResponse, error) {
	query := `SELECT plugin_id, version, configurations, preset_scripts, documentation_url, default_export_url, allow_custom_export_url FROM data_retention_plugin_releases WHERE plugin_id=$1 AND version=$2`
	rows, err := s.db.Queryx(query, req.ID, req.Version)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to fetch plugin")
	}
	defer rows.Close()

	if rows.Next() {
		var plugin RetentionPlugin
		err := rows.StructScan(&plugin)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Failed to read plugin")
		}
		ppb := &pluginpb.GetRetentionPluginConfigResponse{
			Configurations:       plugin.Configurations,
			AllowCustomExportURL: plugin.AllowCustomExportURL,
			PresetScripts:        []*pluginpb.GetRetentionPluginConfigResponse_PresetScript{},
		}
		if plugin.DocumentationURL != nil {
			ppb.DocumentationURL = *plugin.DocumentationURL
		}
		if plugin.DefaultExportURL != nil {
			ppb.DefaultExportURL = *plugin.DefaultExportURL
		}
		if plugin.PresetScripts != nil {
			for _, p := range plugin.PresetScripts {
				ppb.PresetScripts = append(ppb.PresetScripts, &pluginpb.GetRetentionPluginConfigResponse_PresetScript{
					Name:              p.Name,
					Description:       p.Description,
					DefaultFrequencyS: p.DefaultFrequencyS,
					Script:            p.Script,
				})
			}
		}
		return ppb, nil
	}
	return nil, status.Error(codes.NotFound, "plugin not found")
}

// GetRetentionPluginsForOrg gets all data retention plugins enabled by the org.
func (s *Server) GetRetentionPluginsForOrg(ctx context.Context, req *pluginpb.GetRetentionPluginsForOrgRequest) (*pluginpb.GetRetentionPluginsForOrgResponse, error) {
	query := `SELECT r.name, r.id, r.description, r.logo, r.version, r.data_retention_enabled from plugin_releases as r, org_data_retention_plugins as o WHERE r.id = o.plugin_id AND r.version = o.version AND org_id=$1`
	orgID := utils.UUIDFromProtoOrNil(req.OrgID)
	rows, err := s.db.Queryx(query, orgID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to fetch plugin")
	}

	defer rows.Close()

	plugins := []*pluginpb.GetRetentionPluginsForOrgResponse_PluginState{}
	for rows.Next() {
		var p Plugin
		err = rows.StructScan(&p)
		if err != nil {
			return nil, status.Error(codes.Internal, "failed to read plugins")
		}
		ppb := &pluginpb.GetRetentionPluginsForOrgResponse_PluginState{
			Plugin: &pluginpb.Plugin{
				Name:             p.Name,
				ID:               p.ID,
				RetentionEnabled: p.DataRetentionEnabled,
			},
			EnabledVersion: p.Version,
		}
		plugins = append(plugins, ppb)
	}
	return &pluginpb.GetRetentionPluginsForOrgResponse{Plugins: plugins}, nil
}

// GetOrgRetentionPluginConfig gets the org's configuration for a plugin.
func (s *Server) GetOrgRetentionPluginConfig(ctx context.Context, req *pluginpb.GetOrgRetentionPluginConfigRequest) (*pluginpb.GetOrgRetentionPluginConfigResponse, error) {
	query := `SELECT PGP_SYM_DECRYPT(configurations, $1::text), PGP_SYM_DECRYPT(custom_export_url, $1::text) FROM org_data_retention_plugins WHERE org_id=$2 AND plugin_id=$3`

	orgID := utils.UUIDFromProtoOrNil(req.OrgID)
	rows, err := s.db.Queryx(query, s.dbKey, orgID, req.PluginID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to fetch plugin")
	}
	defer rows.Close()

	if rows.Next() {
		var configurationJSON []byte
		var exportURL *string
		var configMap map[string]string

		err := rows.Scan(&configurationJSON, &exportURL)
		if err != nil {
			return nil, status.Error(codes.Internal, "failed to read configs")
		}

		if len(configurationJSON) > 0 {
			err = json.Unmarshal(configurationJSON, &configMap)
			if err != nil {
				return nil, status.Error(codes.Internal, "failed to read configs")
			}
		}

		resp := &pluginpb.GetOrgRetentionPluginConfigResponse{
			Configurations: configMap,
		}

		if exportURL != nil {
			resp.CustomExportUrl = *exportURL
		}

		return resp, nil
	}
	return nil, status.Error(codes.NotFound, "plugin is not enabled")
}

func (s *Server) enableOrgRetention(ctx context.Context, txn *sqlx.Tx, orgID uuid.UUID, pluginID string, version string, configurations []byte, customExportURL *string) error {
	query := `INSERT INTO org_data_retention_plugins (org_id, plugin_id, version, configurations, custom_export_url) VALUES ($1, $2, $3, PGP_SYM_ENCRYPT($4, $5), PGP_SYM_ENCRYPT($6, $5))`
	_, err := txn.Exec(query, orgID, pluginID, version, configurations, s.dbKey, customExportURL)
	if err != nil {
		return status.Errorf(codes.Internal, "Failed to create plugin for org")
	}

	return s.createPresetScripts(ctx, txn, orgID, pluginID, version, configurations, customExportURL)
}

func (s *Server) createPresetScripts(ctx context.Context, txn *sqlx.Tx, orgID uuid.UUID, pluginID string, version string, configurations []byte, customExportURL *string) error {
	// Enabling org retention should enable any preset scripts.
	query := `SELECT preset_scripts FROM data_retention_plugin_releases WHERE plugin_id=$1 AND version=$2`
	rows, err := txn.Queryx(query, pluginID, version)
	if err != nil {
		return status.Errorf(codes.Internal, "Failed to fetch plugin")
	}

	var plugin RetentionPlugin
	if rows.Next() {
		err := rows.StructScan(&plugin)
		if err != nil {
			return status.Errorf(codes.Internal, "Failed to read plugin release")
		}
	}
	rows.Close()

	for _, j := range plugin.PresetScripts {
		_, err = s.createRetentionScript(ctx, txn, orgID, pluginID, &RetentionScript{
			ScriptName:  j.Name,
			Description: j.Description,
			IsPreset:    true,
			ExportURL:   "",
		}, j.Script, make([]*uuidpb.UUID, 0), j.DefaultFrequencyS)
		if err != nil {
			return status.Errorf(codes.Internal, "Failed to create preset scripts")
		}
	}
	return err
}

func (s *Server) disableOrgRetention(ctx context.Context, txn *sqlx.Tx, orgID uuid.UUID, pluginID string) error {
	// Disabling org retention should delete any retention scripts.
	// Fetch all scripts belonging to this plugin.
	query := `DELETE from plugin_retention_scripts WHERE org_id=$1 AND plugin_id=$2 RETURNING script_id`
	rows, err := txn.Queryx(query, orgID, pluginID)
	if err != nil {
		return status.Errorf(codes.Internal, "Failed to fetch scripts")
	}

	for rows.Next() {
		var id uuid.UUID
		err = rows.Scan(&id)
		if err != nil {
			continue
		}
		_, err = s.cronScriptClient.DeleteScript(ctx, &cronscriptpb.DeleteScriptRequest{
			ID: utils.ProtoFromUUID(id),
		})
		if err != nil {
			return status.Errorf(codes.Internal, "Failed to disable script")
		}
	}
	rows.Close()

	query = `DELETE FROM org_data_retention_plugins WHERE org_id=$1 AND plugin_id=$2`
	_, err = txn.Exec(query, orgID, pluginID)
	return err
}

func (s *Server) deletePresetScripts(ctx context.Context, txn *sqlx.Tx, orgID uuid.UUID, pluginID string) error {
	// Disabling org retention should delete any retention scripts.
	// Fetch all scripts belonging to this plugin.
	query := `DELETE from plugin_retention_scripts WHERE org_id=$1 AND plugin_id=$2 AND is_preset=true RETURNING script_id`
	rows, err := txn.Queryx(query, orgID, pluginID)
	if err != nil {
		return status.Errorf(codes.Internal, "Failed to fetch scripts")
	}

	for rows.Next() {
		var id uuid.UUID
		err = rows.Scan(&id)
		if err != nil {
			continue
		}
		_, err = s.cronScriptClient.DeleteScript(ctx, &cronscriptpb.DeleteScriptRequest{
			ID: utils.ProtoFromUUID(id),
		})
		if err != nil {
			return status.Errorf(codes.Internal, "Failed to disable script")
		}
	}
	rows.Close()

	return nil
}

func (s *Server) updateOrgRetentionConfigs(ctx context.Context, txn *sqlx.Tx, orgID uuid.UUID, pluginID string, version string, configurations []byte, customExportURL *string) error {
	query := `UPDATE org_data_retention_plugins SET version = $1, configurations = PGP_SYM_ENCRYPT($2, $3), custom_export_url = PGP_SYM_ENCRYPT($6, $3) WHERE org_id = $4 AND plugin_id = $5`

	err := s.propagateConfigChangesToScripts(ctx, txn, orgID, pluginID, version, configurations, customExportURL)
	if err != nil {
		return err
	}

	_, err = txn.Exec(query, version, configurations, s.dbKey, orgID, pluginID, customExportURL)
	return err
}

func (s *Server) propagateConfigChangesToScripts(ctx context.Context, txn *sqlx.Tx, orgID uuid.UUID, pluginID string, version string, configurations []byte, customExportURL *string) error {
	// Fetch default export URL for plugin.
	pluginExportURL, _, err := s.getPluginConfigs(txn, orgID, pluginID)
	if err != nil {
		return err
	}

	if customExportURL != nil {
		pluginExportURL = *customExportURL
	}

	// Fetch all scripts belonging to this plugin.
	query := `SELECT script_id, PGP_SYM_DECRYPT(export_url, $1::text) as export_url from plugin_retention_scripts WHERE org_id=$2 AND plugin_id=$3`
	rows, err := txn.Queryx(query, s.dbKey, orgID, pluginID)
	if err != nil {
		return status.Errorf(codes.Internal, "Failed to fetch scripts")
	}

	rScripts := make([]*RetentionScript, 0)
	for rows.Next() {
		var rs RetentionScript
		err = rows.StructScan(&rs)
		if err != nil {
			continue
		}
		rScripts = append(rScripts, &rs)
	}
	rows.Close()

	// For each script, update with the new config.
	// TODO(michelle): This is a bit inefficient because we issue a call per script. We should consider adding an RPC method for updating multiple scripts.
	for _, sc := range rScripts {
		var configMap map[string]string
		if len(configurations) != 0 {
			err = json.Unmarshal(configurations, &configMap)
			if err != nil {
				return status.Error(codes.Internal, "failed to read configs")
			}
		}

		exportURL := sc.ExportURL
		if exportURL == "" {
			exportURL = pluginExportURL
		}
		config := &scripts.Config{
			OtelEndpointConfig: &scripts.OtelEndpointConfig{
				URL:     exportURL,
				Headers: configMap,
			},
		}

		mConfig, err := yaml.Marshal(&config)
		if err != nil {
			return status.Error(codes.Internal, "failed to marshal configs")
		}

		_, err = s.cronScriptClient.UpdateScript(ctx, &cronscriptpb.UpdateScriptRequest{
			ScriptId: utils.ProtoFromUUID(sc.ScriptID),
			Configs:  &types.StringValue{Value: string(mConfig)},
		})
		if err != nil {
			log.WithError(err).Error("Failed to update cron script")
			continue
		}
	}

	return nil
}

// UpdateOrgRetentionPluginConfig updates an org's configuration for a plugin.
func (s *Server) UpdateOrgRetentionPluginConfig(ctx context.Context, req *pluginpb.UpdateOrgRetentionPluginConfigRequest) (*pluginpb.UpdateOrgRetentionPluginConfigResponse, error) {
	if utils.IsNilUUIDProto(req.OrgID) {
		return nil, status.Error(codes.InvalidArgument, "Must specify OrgID")
	}

	if req.PluginID == "" {
		return nil, status.Error(codes.InvalidArgument, "Must specify plugin ID")
	}

	if req.Enabled != nil && req.Enabled.Value && req.Version == nil {
		return nil, status.Error(codes.InvalidArgument, "Must specify plugin version when enabling")
	}

	var configurations []byte
	var version string

	orgID := utils.UUIDFromProtoOrNil(req.OrgID)
	if req.Version != nil {
		version = req.Version.Value
	}
	if req.Configurations != nil && len(req.Configurations) > 0 {
		configurations, _ = json.Marshal(req.Configurations)
	}

	txn, err := s.db.Beginx()
	if err != nil {
		return nil, err
	}
	defer txn.Rollback()

	// Fetch current configs.
	query := `SELECT version, PGP_SYM_DECRYPT(configurations, $1::text), PGP_SYM_DECRYPT(custom_export_url, $1::text) FROM org_data_retention_plugins WHERE org_id=$2 AND plugin_id=$3`
	rows, err := txn.Queryx(query, s.dbKey, orgID, req.PluginID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to fetch plugin")
	}

	var origConfig []byte
	var origVersion string
	var customExportURL *string
	enabled := false
	if rows.Next() {
		enabled = true
		err := rows.Scan(&origVersion, &origConfig, &customExportURL)
		if err != nil {
			return nil, status.Error(codes.Internal, "failed to read configs")
		}
	}
	rows.Close()

	if req.CustomExportUrl != nil {
		customExportURL = &req.CustomExportUrl.Value
	}

	ctx, err = contextWithAuthToken(ctx)
	if err != nil {
		return nil, err
	}

	if !enabled && req.Enabled != nil && req.Enabled.Value { // Plugin was just enabled, we should create it.
		err = s.enableOrgRetention(ctx, txn, orgID, req.PluginID, version, configurations, customExportURL)
		if err != nil {
			return nil, err
		}
		return &pluginpb.UpdateOrgRetentionPluginConfigResponse{}, txn.Commit()
	} else if enabled && req.Enabled != nil && !req.Enabled.Value { // Plugin was disabled, we should delete it.
		err = s.disableOrgRetention(ctx, txn, orgID, req.PluginID)
		if err != nil {
			return nil, err
		}
		return &pluginpb.UpdateOrgRetentionPluginConfigResponse{}, txn.Commit()
	} else if !enabled && req.Enabled != nil && !req.Enabled.Value {
		// This is already disabled.
		return &pluginpb.UpdateOrgRetentionPluginConfigResponse{}, nil
	}

	if configurations == nil {
		configurations = origConfig
	}
	if version == "" {
		version = origVersion
	}

	err = s.updateOrgRetentionConfigs(ctx, txn, orgID, req.PluginID, version, configurations, customExportURL)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to update configs")
	}

	if origVersion != version { // The user is updating the plugin, and some of the preset scripts have likely changed.
		// Delete the existing preset scripts and create new ones. However, this means prexisting configurations on scripts will be deleted.
		err := s.deletePresetScripts(ctx, txn, orgID, req.PluginID)
		if err != nil {
			return nil, err
		}

		err = s.createPresetScripts(ctx, txn, orgID, req.PluginID, version, configurations, customExportURL)
		if err != nil {
			return nil, err
		}
	}

	err = txn.Commit()
	if err != nil {
		return nil, err
	}

	return &pluginpb.UpdateOrgRetentionPluginConfigResponse{}, nil
}

// RetentionScript represents a retention script in the plugin system.
type RetentionScript struct {
	OrgID       uuid.UUID `db:"org_id"`
	ScriptID    uuid.UUID `db:"script_id"`
	ScriptName  string    `db:"script_name"`
	Description string    `db:"description"`
	IsPreset    bool      `db:"is_preset"`
	PluginID    string    `db:"plugin_id"`
	ExportURL   string    `db:"export_url"`
}

// GetRetentionScripts gets all retention scripts the org has configured.
func (s *Server) GetRetentionScripts(ctx context.Context, req *pluginpb.GetRetentionScriptsRequest) (*pluginpb.GetRetentionScriptsResponse, error) {
	query := `SELECT r.script_id, r.script_name, r.description, r.is_preset, r.plugin_id from plugin_retention_scripts r, org_data_retention_plugins o WHERE r.org_id=$1 AND r.org_id = o.org_id AND r.plugin_id = o.plugin_id`
	orgID := utils.UUIDFromProtoOrNil(req.OrgID)
	rows, err := s.db.Queryx(query, orgID)
	if err != nil {
		log.WithError(err).Error("Failed to fetch scripts")
		return nil, status.Errorf(codes.Internal, "Failed to fetch scripts")
	}

	defer rows.Close()

	scriptMap := map[uuid.UUID]*pluginpb.RetentionScript{}
	scriptIDs := make([]*uuidpb.UUID, 0)
	for rows.Next() {
		var rs RetentionScript
		err = rows.StructScan(&rs)
		if err != nil {
			return nil, status.Error(codes.Internal, "failed to read script")
		}
		id := utils.ProtoFromUUID(rs.ScriptID)
		scriptMap[rs.ScriptID] = &pluginpb.RetentionScript{
			ScriptID:    id,
			ScriptName:  rs.ScriptName,
			Description: rs.Description,
			PluginId:    rs.PluginID,
			IsPreset:    rs.IsPreset,
		}
		scriptIDs = append(scriptIDs, id)
	}

	if len(scriptIDs) == 0 {
		return &pluginpb.GetRetentionScriptsResponse{Scripts: make([]*pluginpb.RetentionScript, 0)}, nil
	}

	ctx, err = contextWithAuthToken(ctx)
	if err != nil {
		return nil, err
	}
	cronScriptsResp, err := s.cronScriptClient.GetScripts(ctx, &cronscriptpb.GetScriptsRequest{IDs: scriptIDs})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to fetch cron scripts")
	}

	for _, c := range cronScriptsResp.Scripts {
		if v, ok := scriptMap[utils.UUIDFromProtoOrNil(c.ID)]; ok {
			v.FrequencyS = c.FrequencyS
			v.Enabled = c.Enabled
			v.ClusterIDs = c.ClusterIDs
		}
	}

	scripts := make([]*pluginpb.RetentionScript, 0)
	for _, v := range scriptMap {
		scripts = append(scripts, v)
	}
	return &pluginpb.GetRetentionScriptsResponse{Scripts: scripts}, nil
}

// GetRetentionScript gets the details for a script an org is using for long-term data retention.
func (s *Server) GetRetentionScript(ctx context.Context, req *pluginpb.GetRetentionScriptRequest) (*pluginpb.GetRetentionScriptResponse, error) {
	query := `SELECT script_name, description, is_preset, plugin_id, PGP_SYM_DECRYPT(export_url, $1::text) as export_url from plugin_retention_scripts WHERE org_id=$2 AND script_id=$3`
	orgID := utils.UUIDFromProtoOrNil(req.OrgID)
	scriptID := utils.UUIDFromProtoOrNil(req.ScriptID)
	rows, err := s.db.Queryx(query, s.dbKey, orgID, scriptID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to fetch script")
	}

	defer rows.Close()

	if !rows.Next() {
		return nil, status.Error(codes.NotFound, "script not found")
	}

	var script RetentionScript
	err = rows.StructScan(&script)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to read script")
	}

	ctx, err = contextWithAuthToken(ctx)
	if err != nil {
		return nil, err
	}

	cronScriptResp, err := s.cronScriptClient.GetScript(ctx, &cronscriptpb.GetScriptRequest{ID: req.ScriptID})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to fetch cron script")
	}
	cronScript := cronScriptResp.Script

	return &pluginpb.GetRetentionScriptResponse{
		Script: &pluginpb.DetailedRetentionScript{
			Script: &pluginpb.RetentionScript{
				ScriptID:    req.ScriptID,
				ScriptName:  script.ScriptName,
				Description: script.Description,
				FrequencyS:  cronScript.FrequencyS,
				ClusterIDs:  cronScript.ClusterIDs,
				PluginId:    script.PluginID,
				Enabled:     cronScript.Enabled,
				IsPreset:    script.IsPreset,
			},
			Contents:  cronScript.Script,
			ExportURL: script.ExportURL,
		},
	}, nil
}

func (s *Server) createRetentionScript(ctx context.Context, txn *sqlx.Tx, orgID uuid.UUID, pluginID string, rs *RetentionScript, contents string, clusterIDs []*uuidpb.UUID, frequencyS int64) (*uuidpb.UUID, error) {
	pluginExportURL, configMap, err := s.getPluginConfigs(txn, orgID, pluginID)
	if err != nil {
		return nil, err
	}

	exportURL := pluginExportURL
	if rs.ExportURL != "" {
		exportURL = rs.ExportURL
	}

	configYAML, err := scriptConfigToYAML(configMap, exportURL)
	if err != nil {
		return nil, err
	}

	cronScriptResp, err := s.cronScriptClient.CreateScript(ctx, &cronscriptpb.CreateScriptRequest{
		Script:     contents,
		ClusterIDs: clusterIDs,
		Configs:    configYAML,
		FrequencyS: frequencyS,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to create cron script")
	}

	scriptID := cronScriptResp.ID

	query := `INSERT INTO plugin_retention_scripts (org_id, plugin_id, script_id, script_name, description, export_url, is_preset) VALUES ($1, $2, $3, $4, $5, PGP_SYM_ENCRYPT($6, $7), $8)`
	_, err = txn.Exec(query, orgID, pluginID, utils.UUIDFromProtoOrNil(scriptID), rs.ScriptName, rs.Description, rs.ExportURL, s.dbKey, rs.IsPreset)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to create retention script")
	}
	return scriptID, nil
}

// CreateRetentionScript creates a script that is used for long-term data retention.
func (s *Server) CreateRetentionScript(ctx context.Context, req *pluginpb.CreateRetentionScriptRequest) (*pluginpb.CreateRetentionScriptResponse, error) {
	orgID := utils.UUIDFromProtoOrNil(req.OrgID)
	txn, err := s.db.Beginx()
	if err != nil {
		return nil, err
	}
	defer txn.Rollback()

	ctx, err = contextWithAuthToken(ctx)
	if err != nil {
		return nil, err
	}

	id, err := s.createRetentionScript(ctx, txn, orgID, req.Script.Script.PluginId, &RetentionScript{
		ScriptName:  req.Script.Script.ScriptName,
		Description: req.Script.Script.Description,
		IsPreset:    req.Script.Script.IsPreset,
		ExportURL:   req.Script.ExportURL,
	}, req.Script.Contents, req.Script.Script.ClusterIDs, req.Script.Script.FrequencyS)
	if err != nil {
		return nil, err
	}

	err = txn.Commit()
	if err != nil {
		return nil, err
	}

	return &pluginpb.CreateRetentionScriptResponse{
		ID: id,
	}, nil
}

func (s *Server) getPluginConfigs(txn *sqlx.Tx, orgID uuid.UUID, pluginID string) (string, map[string]string, error) {
	query := `SELECT PGP_SYM_DECRYPT(o.configurations, $1::text), r.default_export_url, PGP_SYM_DECRYPT(o.custom_export_url, $1::text) FROM org_data_retention_plugins o, data_retention_plugin_releases r WHERE org_id=$2 AND r.plugin_id=$3 AND o.plugin_id=r.plugin_id AND r.version = o.version`
	rows, err := txn.Queryx(query, s.dbKey, orgID, pluginID)
	if err != nil {
		return "", nil, status.Errorf(codes.Internal, "failed to fetch plugin")
	}
	defer rows.Close()

	if !rows.Next() {
		return "", nil, status.Error(codes.NotFound, "plugin is not enabled")
	}

	var configurationJSON []byte
	var configMap map[string]string
	var pluginExportURL string
	var customExportURL *string
	err = rows.Scan(&configurationJSON, &pluginExportURL, &customExportURL)
	if err != nil {
		return "", nil, status.Error(codes.Internal, "failed to read configs")
	}
	if len(configurationJSON) > 0 {
		err = json.Unmarshal(configurationJSON, &configMap)
		if err != nil {
			return "", nil, status.Error(codes.Internal, "failed to read configs")
		}
	}

	if customExportURL != nil {
		pluginExportURL = *customExportURL
	}

	return pluginExportURL, configMap, nil
}

func scriptConfigToYAML(configMap map[string]string, exportURL string) (string, error) {
	config := &scripts.Config{
		OtelEndpointConfig: &scripts.OtelEndpointConfig{
			URL:     exportURL,
			Headers: configMap,
		},
	}

	mConfig, err := yaml.Marshal(&config)
	if err != nil {
		return "", status.Error(codes.Internal, "failed to marshal configs")
	}
	return string(mConfig), nil
}

// UpdateRetentionScript updates a script used for long-term data retention.
func (s *Server) UpdateRetentionScript(ctx context.Context, req *pluginpb.UpdateRetentionScriptRequest) (*pluginpb.UpdateRetentionScriptResponse, error) {
	txn, err := s.db.Beginx()
	if err != nil {
		return nil, err
	}
	defer txn.Rollback()

	ctx, err = contextWithAuthToken(ctx)
	if err != nil {
		return nil, err
	}

	// Fetch existing script.
	query := `SELECT org_id, script_name, description, PGP_SYM_DECRYPT(export_url, $1::text) as export_url, plugin_id from plugin_retention_scripts WHERE script_id=$2`
	scriptID := utils.UUIDFromProtoOrNil(req.ScriptID)
	rows, err := txn.Queryx(query, s.dbKey, scriptID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to fetch script")
	}

	if !rows.Next() {
		return nil, status.Error(codes.NotFound, "script not found")
	}

	var script RetentionScript
	err = rows.StructScan(&script)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to read script")
	}
	rows.Close()

	sCtx, err := authcontext.FromContext(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "Unauthenticated")
	}
	claimsOrgIDstr := sCtx.Claims.GetUserClaims().OrgID
	if script.OrgID.String() != claimsOrgIDstr {
		return nil, status.Errorf(codes.Unauthenticated, "Unauthorized")
	}

	// Fetch config + headers from plugin info.
	pluginExportURL, configMap, err := s.getPluginConfigs(txn, script.OrgID, script.PluginID)
	if err != nil {
		return nil, err
	}
	scriptName := script.ScriptName
	description := script.Description
	exportURL := script.ExportURL

	if req.ScriptName != nil {
		scriptName = req.ScriptName.Value
	}
	if req.Description != nil {
		description = req.Description.Value
	}
	if req.ExportUrl != nil {
		exportURL = req.ExportUrl.Value
	}

	// Update retention scripts with new info.
	query = `UPDATE plugin_retention_scripts SET script_name = $1, export_url = PGP_SYM_ENCRYPT($2, $3), description = $4 WHERE script_id = $5`
	_, err = txn.Exec(query, scriptName, exportURL, s.dbKey, description, scriptID)

	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to update retention script")
	}
	err = txn.Commit()
	if err != nil {
		return nil, err
	}

	// Create updated config with new export URL.
	configExportURL := exportURL
	if exportURL == "" {
		configExportURL = pluginExportURL
	}
	configYAML, err := scriptConfigToYAML(configMap, configExportURL)
	if err != nil {
		return nil, err
	}

	// Update cron script.
	_, err = s.cronScriptClient.UpdateScript(ctx, &cronscriptpb.UpdateScriptRequest{
		Script:     req.Contents,
		ClusterIDs: &cronscriptpb.ClusterIDs{Value: req.ClusterIDs},
		Enabled:    req.Enabled,
		FrequencyS: req.FrequencyS,
		ScriptId:   req.ScriptID,
		Configs:    &types.StringValue{Value: configYAML},
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to update cron script")
	}

	return &pluginpb.UpdateRetentionScriptResponse{}, nil
}

// DeleteRetentionScript creates a script that is used for long-term data retention.
func (s *Server) DeleteRetentionScript(ctx context.Context, req *pluginpb.DeleteRetentionScriptRequest) (*pluginpb.DeleteRetentionScriptResponse, error) {
	txn, err := s.db.Beginx()
	if err != nil {
		return nil, err
	}
	defer txn.Rollback()

	orgID := utils.UUIDFromProtoOrNil(req.OrgID)
	scriptID := utils.UUIDFromProtoOrNil(req.ID)

	query := `DELETE FROM plugin_retention_scripts WHERE org_id=$1 AND script_id=$2 AND NOT is_preset`
	resp, err := txn.Exec(query, orgID, scriptID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to delete scripts")
	}

	rowsDel, err := resp.RowsAffected()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to delete scripts")
	}
	if rowsDel == 0 {
		return nil, status.Errorf(codes.Internal, "No script to delete")
	}

	ctx, err = contextWithAuthToken(ctx)
	if err != nil {
		return nil, err
	}

	_, err = s.cronScriptClient.DeleteScript(ctx, &cronscriptpb.DeleteScriptRequest{
		ID: req.ID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to delete cron script")
	}

	err = txn.Commit()
	if err != nil {
		return nil, err
	}

	return &pluginpb.DeleteRetentionScriptResponse{}, nil
}
