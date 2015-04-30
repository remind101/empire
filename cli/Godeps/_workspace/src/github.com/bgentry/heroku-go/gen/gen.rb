#!/usr/bin/env ruby

require 'erubis'
require 'multi_json'

RESOURCE_TEMPLATE = <<-RESOURCE_TEMPLATE
// WARNING: This code is auto-generated from the Heroku Platform API JSON Schema
// by a Ruby script (gen/gen.rb). Changes should be made to the generation
// script rather than the generated files.

package heroku

<%- if schemas[key]['properties'] && schemas[key]['properties'].any?{|p, v| resolve_typedef(v).end_with?("time.Time") } %>
import (
	"time"
)
<%- end %>

<%- if definition['properties'] %>
  <%- description = markdown_free(definition["description"] || "") %>
  <%- word_wrap(description, line_width: 77).split("\n").each do |line| %>
    // <%= line %>
  <%- end %>
  type <%= resource_class %> struct {
  <%- definition['properties'].each do |propname, propdef| %>
    <%- resolved_propdef = resolve_propdef(propdef) %>
    // <%= resolved_propdef["description"] %>
    <%- type = resolve_typedef(resolved_propdef) %>
    <%= Erubis::Eruby.new(STRUCT_FIELD_TEMPLATE).result({type: type, propdef: propdef, resolved_propdef: resolved_propdef, propname: propname}).strip %>

  <%- end %>
  }
<%- end %>

<%- definition["links"].each do |link| %>
  <%- func_name = titlecase(key.downcase. + "-" + link["title"]) %>
  <%- func_args = [] %>
  <%- func_args += func_args_from_model_and_link(definition, key, link) %>
  <%- return_values = return_values_from_link(key, link) %>
  <%- path = link['href'] %>
  <%- if parent_resource_instance %>
    <%- path = path.gsub("{(%23%2Fdefinitions%2F" + parent_resource_instance + "%2Fdefinitions%2Fidentity)}", '" + ' + variablecase(parent_resource_instance) + 'Identity + "') %>
  <%- end %>
  <%- path = substitute_resource_identity(path) %>
  <%- path = ensure_balanced_end_quote(ensure_open_quote(path)) %>

  <%- word_wrap(markdown_free(link["description"]), line_width: 77).split("\n").each do |line| %>
    // <%= line %>
  <%- end %>
  <%- func_arg_comments = [] %>
  <%- func_arg_comments += func_arg_comments_from_model_and_link(definition, key, link) %>
  <%- unless func_arg_comments.empty? %>
    //
  <%- end %>
  <%- word_wrap(func_arg_comments.join(" "), line_width: 77).split("\n").each do |comment| %>
    // <%= comment %>
  <%- end %>
  <%- flat_postval = link["schema"] && link["schema"]["additionalProperties"] == false %>
  <%- required = (link["schema"] && link["schema"]["required"]) || [] %>
  <%- optional = ((link["schema"] && link["schema"]["properties"]) || {}).keys - required %>
  <%- postval = if flat_postval %>
    <%-           "options" %>
    <%-         elsif required.empty? && optional.empty? %>
    <%-           "nil" %>
    <%-         elsif required.empty? %>
    <%-           "options" %>
    <%-         else %>
    <%-           "params" %>
    <%-         end %>
  <%- hasCustomType = !schemas[key]["properties"].nil? %>
  func (c *Client) <%= func_name + "(" + func_args.join(', ') %>) <%= return_values %> {
    <%- method = link['method'].downcase.capitalize %>
    <%- case link["rel"] %>
    <%- when "create" %>
      <%- if !required.empty? %>
        <%= Erubis::Eruby.new(LINK_PARAMS_TEMPLATE).result({modelname: key, link: link, required: required, optional: optional}).strip %>
      <%- end %>
      var <%= variablecase(key + '-res') %> <%= titlecase(key) %>
      <%- puts link if key =~ /org/ %>
      return &<%= variablecase(key + '-res') %>, c.<%= method %>(&<%= variablecase(key + '-res') %>, <%= path %>, <%= postval %>)
    <%- when "self" %>
      var <%= variablecase(key) %> <%= hasCustomType ? titlecase(key) : "map[string]string" %>
      return <%= "&" if hasCustomType%><%= variablecase(key) %>, c.<%= method %>(&<%= variablecase(key) %>, <%= path %>)
    <%- when "destroy", "empty" %>
      return c.Delete(<%= path %>)
    <%- when "update" %>
      <%- if !required.empty? %>
        <%= Erubis::Eruby.new(LINK_PARAMS_TEMPLATE).result({modelname: key, link: link, required: required, optional: optional}).strip %>
      <%- end %>
      <%- if link["title"].include?("Batch") %>
        var <%= variablecase(key + 's-res') %> []<%= titlecase(key) %>
        return <%= variablecase(key + 's-res') %>, c.<%= method %>(&<%= variablecase(key + 's-res') %>, <%= path %>, <%= postval %>)
      <%- else %>
        var <%= variablecase(key + '-res') %> <%= hasCustomType ? titlecase(key) : "map[string]string" %>
        return <%= "&" if hasCustomType%><%= variablecase(key + '-res') %>, c.<%= method %>(&<%= variablecase(key + '-res') %>, <%= path %>, <%= postval %>)
      <%- end %>
    <%- when "instances" %>
      req, err := c.NewRequest("GET", <%= path %>, nil)
      if err != nil {
        return nil, err
      }

      if lr != nil {
        lr.SetHeader(req)
      }

      var <%= variablecase(key + 's-res') %> []<%= titlecase(key) %>
      return <%= variablecase(key + 's-res') %>, c.DoReq(req, &<%= variablecase(key + 's-res') %>)
    <%- end %>
  }

  <%- if %w{create update}.include?(link["rel"]) && link["schema"] && link["schema"]["properties"] %>
    <%- if !required.empty? %>
      <%- structs = required.select {|p| resolve_typedef(link["schema"]["properties"][p]) == "struct" } %>
      <%- structs.each do |propname| %>
        <%- typename = titlecase([key, link["title"], propname].join("-")) %>
        // <%= typename %> used in <%= func_name %> as the <%= definition["properties"][propname]["description"] %>
        type <%= typename %> struct {
          <%- link["schema"]["properties"][propname]["properties"].each do |subpropname, subval| %>
            <%- propdef = definition["properties"][propname]["properties"][subpropname] %>
            <%- description = resolve_propdef(propdef)["description"] %>
            <%- word_wrap(description, line_width: 77).split("\n").each do |line| %>
              // <%= line %>
            <%- end %>
            <%= titlecase(subpropname) %> <%= resolve_typedef(subval) %> `json:"<%= subpropname %>"`

          <%- end %>
        }
      <%- end %>
      <%- arr_structs = required.select {|p| resolve_typedef(link["schema"]["properties"][p]) == "[]struct" } %>
      <%- arr_structs.each do |propname| %>
        <%- # special case for arrays of structs (like FormationBulkUpdate) %>
        <%- typename = titlecase([key, link["title"], "opts"].join("-")) %>
        <%- typedef = resolve_propdef(link["schema"]["properties"][propname]["items"]) %>

        type <%= typename %> struct {
          <%- typedef["properties"].each do |subpropname, subref| %>
            <%- propdef = resolve_propdef(subref) %>
            <%- description = resolve_propdef(propdef)["description"] %>
            <%- is_required = typedef["required"].include?(subpropname) %>
            <%- word_wrap(description, line_width: 77).split("\n").each do |line| %>
              // <%= line %>
            <%- end %>
            <%= titlecase(subpropname) %> <%= "*" unless is_required %><%= resolve_typedef(propdef) %> `json:"<%= subpropname %><%= ",omitempty" unless is_required %>"`

          <%- end %>
        }
      <%- end %>
    <%- end %>
    <%- if !optional.empty? %>
      // <%= func_name %>Opts holds the optional parameters for <%= func_name %>
      type <%= func_name %>Opts struct {
        <%- optional.each do |propname| %>
          <%- if definition['properties'][propname] && definition['properties'][propname]['description'] %>
            // <%= definition['properties'][propname]['description'] %>
          <%- elsif definition["definitions"][propname] %>
            // <%= definition["definitions"][propname]["description"] %>
          <%- elsif link["schema"]["properties"][propname]["$ref"] %>
            // <%= resolve_propdef(link["schema"]["properties"][propname])["description"] %>
          <%- else %>
            // <%= link["schema"]["properties"][propname]["description"] %>
          <%- end %>
          <%= titlecase(propname) %> <%= type_for_link_opts_field(link, propname) %> `json:"<%= propname %>,omitempty"`
        <%- end %>
      }
    <%- end %>
  <%- end %>

<%- end %>
RESOURCE_TEMPLATE

STRUCT_FIELD_TEMPLATE = <<-STRUCT_FIELD_TEMPLATE
<%- if type =~ /\\*?struct/ %>
  <%- is_array = propdef["type"].is_a?(Array) && propdef["type"].first == "array" %>
  <%= titlecase(propname) %> <%= "[]" if is_array %><%= type %> {
    <%- resolved_propdef["properties"].each do |subpropname, subpropdef| %>
      <%- resolved_subpropdef = resolve_propdef(subpropdef) %>
      <%- subtype = resolve_typedef(resolved_subpropdef) %>
      <%- if subtype =~ /\\*?struct/ %>
        <%= Erubis::Eruby.new(STRUCT_FIELD_TEMPLATE).result({type: subtype, propdef: subpropdef, resolved_propdef: resolved_subpropdef, propname: subpropname}).strip %>
      <%- else %>
        <%= Erubis::Eruby.new(FLAT_STRUCT_FIELD_TEMPLATE).result({type: subtype, propname: subpropname}).strip %>
      <%- end %>
    <%- end %>
  } `json:"<%= propname %>"`
<%- else %>
  <%= Erubis::Eruby.new(FLAT_STRUCT_FIELD_TEMPLATE).result({type: type, propname: propname}).strip %>
<%- end %>
STRUCT_FIELD_TEMPLATE

FLAT_STRUCT_FIELD_TEMPLATE = <<-FLAT_STRUCT_FIELD_TEMPLATE
  <%= titlecase(propname) %> <%= type %> `json:"<%= propname %>"`
FLAT_STRUCT_FIELD_TEMPLATE

LINK_PARAMS_TEMPLATE = <<-LINK_PARAMS_TEMPLATE
params := struct {
<%- required.each do |propname| %>
  <%- type = resolve_typedef(link["schema"]["properties"][propname]) %>
  <%- if type == "[]struct" %>
    <%- type = type.gsub("struct", titlecase([modelname, link["title"], "opts"].join("-"))) %>
  <%- elsif type == "struct" %>
    <%- type = titlecase([modelname, link["title"], propname].join("-")) %>
  <%- end %>
  <%= titlecase(propname) %> <%= type %> `json:"<%= propname %>"`
<%- end %>
<%- optional.each do |propname| %>
  <%= titlecase(propname) %> <%= type_for_link_opts_field(link, propname) %> `json:"<%= propname %>,omitempty"`
<%- end %>
}{
<%- required.each do |propname| %>
  <%= titlecase(propname) %>: <%= variablecase(propname) %>,
<%- end %>
}
<%- if optional.count > 0 %>
  if options != nil {
  <%- optional.each do |propname| %>
    params.<%= titlecase(propname) %> = options.<%= titlecase(propname) %>
  <%- end %>
  }
<%- end %>
LINK_PARAMS_TEMPLATE

#   definition:               data,
#   key:                      modelname,
#   parent_resource_class:    parent_resource_class,
#   parent_resource_identity: parent_resource_identity,
#   parent_resource_instance: parent_resource_instance,
#   resource_class:           resource_class,
#   resource_instance:        resource_instance,
#   resource_proxy_class:     resource_proxy_class,
#   resource_proxy_instance:  resource_proxy_instance

module Generator
  extend self

  def ensure_open_quote(str)
    str[0] == '"' ? str : "\"#{str}"
  end

  def ensure_balanced_end_quote(str)
    (str.count('"') % 2) == 1 ? "#{str}\"" : str
  end

  def strip_unnecessary_end_plusquote(str)
    str.gsub(/ \+ "$/, "")
  end

  def must_end_with(str, ending)
    str.end_with?(ending) ? str : "#{str}#{ending}"
  end

  def word_wrap(text, options = {})
    line_width = options.fetch(:line_width, 80)

    text.split("\n").collect do |line|
      line.length > line_width ? line.gsub(/(.{1,#{line_width}})(\s+|$)/, "\\1\n").strip : line
    end * "\n"
  end

  def markdown_free(text)
    text.gsub(/\[(?<linktext>[^\]]*)\](?<linkurl>\(.*\))/, '\k<linktext>').
      gsub(/`(?<rawtext>[^\]]*)`/, '\k<rawtext>').gsub("NULL", "nil")
  end

  def variablecase(str)
    words = str.gsub('_','-').gsub(' ','-').split('-')
    (words[0...1] + words[1..-1].map {|k| k[0...1].upcase + k[1..-1]}).join
  end

  def titlecase(str)
    str.gsub('_','-').gsub(' ','-').split('-').map do |k|
      # special case so Url becomes URL, Ssl becomes SSL
      if %w{url ssl}.include?(k.downcase)
        k.upcase
      elsif k.downcase == "oauth" # special case so Oauth becomes OAuth
        "OAuth"
      else
        k[0...1].upcase + k[1..-1]
      end
    end.join
  end

  def resolve_typedef(propdef)
    if types = propdef["type"]
      null = types.include?("null")
      tname = case (types - ["null"]).first
              when "boolean"
                "bool"
              when "integer"
                "int"
              when "string"
                format = propdef["format"]
                format && format == "date-time" ? "time.Time" : "string"
              when "object"
                if propdef["additionalProperties"] == false
                  if propdef["patternProperties"]
                    "map[string]string"
                  else
                    # special case for arrays of structs (like FormationBulkUpdate)
                    "struct"
                  end
                else
                  "struct"
                end
              when "array"
                arraytype = if propdef["items"]["$ref"]
                  resolve_typedef(propdef["items"])
                else
                  propdef["items"]["type"]
                end
                arraytype = arraytype.first if arraytype.is_a?(Array)
                "[]#{arraytype}"
              else
                types.first
              end
      null ? "*#{tname}" : tname
    elsif propdef["anyOf"]
      # identity cross-reference, cheat because these are always strings atm
      "string"
    elsif propdef["additionalProperties"] == false
      # inline object
      propdef
    elsif ref = propdef["$ref"]
      matches = ref.match(/\/definitions\/([\w-]+)\/definitions\/([\w-]+)/)
      schemaname, fieldname = matches[1..2]
      resolve_typedef(schemas[schemaname]["definitions"][fieldname])
    else
      raise "WTF #{propdef}"
    end
  end

  def type_for_link_opts_field(link, propname, nullable = true)
    resulttype = resolve_typedef(link["schema"]["properties"][propname])
    if nullable && !resulttype.start_with?("*")
      resulttype = "*#{resulttype}"
    elsif !nullable
      resulttype = resulttype.gsub("*", "")
    end
    resulttype
  end

  def type_from_types_and_format(types, format)
    case types.first
    when "boolean"
      "bool"
    when "integer"
      "int"
    when "string"
      format && format == "date-time" ? "time.Time" : "string"
    else
      types.first
    end
  end

  def return_values_from_link(modelname, link)
    if !schemas[modelname]["properties"]
      # structless type like ConfigVar
      "(map[string]string, error)"
    else
      case link["rel"]
      when "destroy", "empty"
        "error"
      when "instances"
        "([]#{titlecase(modelname)}, error)"
      else
        if link["title"].include?("Batch")
          "([]#{titlecase(modelname)}, error)"
        else
          "(*#{titlecase(modelname)}, error)"
        end
      end
    end
  end

  def func_args_from_model_and_link(definition, modelname, link)
    args = []
    required = (link["schema"] && link["schema"]["required"]) || []
    optional = ((link["schema"] && link["schema"]["properties"]) || {}).keys - required

    # get all of the model identities required by this link's href
    reg = /{\(%23%2Fdefinitions%2F(?<keyname>[\w-]+)%2Fdefinitions%2Fidentity\)}/
    link["href"].scan(reg) do |match|
      args << "#{variablecase(match.first)}Identity string"
    end

    if %w{create update}.include?(link["rel"])
      if link["schema"]["additionalProperties"] == false
        # handle ConfigVar update
        args << "options map[string]*string"
      else
        required.each do |propname|
          type = type_for_link_opts_field(link, propname, false)
          if type == "[]struct"
            type = type.gsub("struct", titlecase([modelname, link["title"], "Opts"].join("-")))
          elsif type == "struct"
            type = type.gsub("struct", titlecase([modelname, link["title"], propname].join("-")))
          end
          args << "#{variablecase(propname)} #{type}"
        end
      end
      args << "options *#{titlecase(modelname)}#{link["rel"].capitalize}Opts" unless optional.empty?
    end

    if "instances" == link["rel"]
      args << "lr *ListRange"
    end

    args
  end

  def resolve_propdef(propdef)
    resolve_all_propdefs(propdef).first
  end

  def resolve_all_propdefs(propdef)
    if propdef["description"]
      [propdef]
    elsif ref = propdef["$ref"]
      # handle embedded structs
      if matches = ref.match(/#\/definitions\/([\w-]+)\/definitions\/([\w-]+)\/definitions\/([\w-]+)/)
        schemaname, structname, fieldname = matches[1..3]
        resolve_all_propdefs(schemas[schemaname]["definitions"][structname]["definitions"][fieldname])
      else
        matches = ref.match(/#\/definitions\/([\w-]+)\/definitions\/([\w-]+)/)
        schemaname, fieldname = matches[1..2]
        resolve_all_propdefs(schemas[schemaname]["definitions"][fieldname])
      end
    elsif anyof = propdef["anyOf"]
      # Identity
      anyof.map do |refhash|
        matches = refhash["$ref"].match(/#\/definitions\/([\w-]+)\/definitions\/([\w-]+)/)
        schemaname, fieldname = matches[1..2]
        resolve_all_propdefs(schemas[schemaname]["definitions"][fieldname])
      end.flatten
    elsif propdef["items"] && ref = propdef["items"]["$ref"]
      # special case for params which are embedded structs, like build-result
      matches = ref.match(/#\/definitions\/([\w-]+)\/definitions\/([\w-]+)/)
      schemaname, fieldname = matches[1..2]
      resolve_all_propdefs(schemas[schemaname]["definitions"][fieldname])
    elsif propdef["type"] && propdef["type"].is_a?(Array) && propdef["type"].first == "object"
      # special case for params which are nested objects, like oauth-grant
      [propdef]
    else
      raise "WTF #{propdef}"
    end
  end

  def func_arg_comments_from_model_and_link(definition, modelname, link)
#   <%- func_arg_comments << (variablecase(parent_resource_instance) + "Identity is the unique identifier of the " + key + "'s " + parent_resource_instance + ".") if parent_resource_instance %>
    args = []
    flat_postval = link["schema"] && link["schema"]["additionalProperties"] == false
    properties = (link["schema"] && link["schema"]["properties"]) || {}
    required_keys = (link["schema"] && link["schema"]["required"]) || []
    optional_keys = properties.keys - required_keys

    # get all of the model identities required by this link's href
    reg = /{\(%23%2Fdefinitions%2F(?<keyname>[\w-]+)%2Fdefinitions%2Fidentity\)}/
    link["href"].scan(reg) do |match|
      if match.first == modelname
        args << "#{variablecase(modelname)}Identity is the unique identifier of the #{titlecase(modelname)}."
      else
        args << "#{variablecase(match.first)}Identity is the unique identifier of the #{titlecase(modelname)}'s #{titlecase(match.first)}."
      end
    end

    if flat_postval
      # special case for ConfigVar update w/ flat param struct
      desc = markdown_free(link["schema"]["description"])
      args << "options is the #{desc}."
    end

    if %w{create update}.include?(link["rel"])
      required_keys.each do |propname|
        rpresults = resolve_all_propdefs(link["schema"]["properties"][propname])
        if rpresults.size == 1
          if rpresults.first["properties"]
            # special case for things like OAuthToken with nested objects
            rpresults = resolve_all_propdefs(definition["properties"][propname])
          end
          args << "#{variablecase(propname)} is the #{must_end_with(rpresults.first["description"] || "", ".")}"
        elsif rpresults.size == 2
          args << "#{variablecase(propname)} is the #{rpresults.first["description"]} or #{must_end_with(rpresults.last["description"] || "", ".")}"
        else
          raise "Didn't expect 3 rpresults"
        end
      end
      args << "options is the struct of optional parameters for this action." unless optional_keys.empty?
    end

    if "instances" == link["rel"]
      args << "lr is an optional ListRange that sets the Range options for the paginated list of results."
    end

    case link["rel"]
    when "create"
      ["options is the struct of optional parameters for this action."]
    when "update"
      ["#{variablecase(modelname)}Identity is the unique identifier of the #{titlecase(modelname)}.",
       "options is the struct of optional parameters for this action."]
    when "destroy", "self", "empty"
      ["#{variablecase(modelname)}Identity is the unique identifier of the #{titlecase(modelname)}."]
    when "instances"
      ["lr is an optional ListRange that sets the Range options for the paginated list of results."]
    else
      []
    end
    args
  end

  def substitute_resource_identity(path)
    reg = /{\(%23%2Fdefinitions%2F(?<keyname>[\w-]+)%2Fdefinitions%2Fidentity\)}/
    matches = path.match(reg)
    return path unless matches

    gsubbed = path.gsub(reg, '"+' + variablecase(matches['keyname']) + 'Identity + "')
    strip_unnecessary_end_plusquote(gsubbed)
  end

  def resource_instance_from_model(modelname)
    modelname.downcase.split('-').join('_')
  end

  def schemas
    @@schemas ||= {}
  end

  def load_schema
    schema_path = File.expand_path("./schema.json")
    schema = MultiJson.load(File.read(schema_path))
    schema["definitions"].each do |modelname, val|
      schemas[modelname] = val
    end
  end

  def generate_model(modelname)
    if !schemas[modelname]
      puts "no schema for #{modelname}" && return
    end
    if schemas[modelname]['links'].empty?
      puts "no links for #{modelname}"
    end

    resource_class = titlecase(modelname)
    resource_instance = resource_instance_from_model(modelname)

    resource_proxy_class = resource_class + 's'
    resource_proxy_instance = resource_instance + 's'

    parent_resource_class, parent_resource_identity, parent_resource_instance = if schemas[modelname]['links'].all? {|link| link['href'].include?('{(%23%2Fdefinitions%2Fapp%2Fdefinitions%2Fidentity)}')}
      ['App', 'app_identity', 'app']
    elsif schemas[modelname]['links'].all? {|link| link['href'].include?('{(%23%2Fdefinitions%2Faddon-service%2Fdefinitions%2Fidentity)}')}
      ['AddonService', 'addon_service', 'addon-service']
    end

    data = Erubis::Eruby.new(RESOURCE_TEMPLATE).result({
      definition:               schemas[modelname],
      key:                      modelname,
      parent_resource_class:    parent_resource_class,
      parent_resource_identity: parent_resource_identity,
      parent_resource_instance: parent_resource_instance,
      resource_class:           resource_class,
      resource_instance:        resource_instance,
      resource_proxy_class:     resource_proxy_class,
      resource_proxy_instance:  resource_proxy_instance
    })

    path = File.expand_path(File.join(File.dirname(__FILE__), '..', "#{modelname.gsub('-', '_')}.go"))
    File.open(path, 'w') do |file|
      file.write(data)
    end
    %x( go fmt #{path} )
  end
end

include Generator

puts "Loading schema..."
Generator.load_schema

schemas.keys.each do |modelname|
  puts "Generating #{modelname}..."
  if (Generator.schemas[modelname]["links"] || []).empty? && Generator.schemas[modelname]["properties"].empty?
    puts "-- skipping #{modelname} because it has no links or properties"
  else
    Generator.generate_model(modelname)
  end
end
