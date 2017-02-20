setwd('/Users/rlan/Work/go/src/fourfourtwo/')

library(ggplot2)
library(RSQLite)
library(dplyr)
library(sqldf)
library(stringr)

##########################################
#     Define Functions 
#########################################
circleFun <- function(center = c(0,0),diameter = 1, npoints = 100){
  r = diameter / 2
  tt <- seq(0,2*pi,length.out = npoints)
  xx <- center[1] + r * cos(tt)
  yy <- center[2] + r * sin(tt)
  return(data.frame(x = xx, y = yy))
}

draw_scatter_plot_of_player_activity <- function (df, base_dir) {
  # some_magic_number:
  # boundary: 51, 51, 689, 479
  # outer_box: x: +-99 (based on boundary), y: +-86 (based on boundary)
  # inner_box: x: +-34 (based on boundary), y: +-71 (based on outer_box)
  # gate: y: +-36 (based on horizontal middle line), gate_depth: 20
  
  pitch_boundary <- data.frame(x=c(51,51,689,689,51), y=c(51,479,479,51,51))
  pitch_left_outer_box <- data.frame(x=c(51,150,150,51), y=c(393,393,137,137))
  pitch_right_outer_box <- data.frame(x=c(689,590,590,689), y=c(393,393,137,137))
  pitch_left_inner_box <- data.frame(x=c(51,85,85,51), y=c(322,322,208,208))
  pitch_right_inner_box <- data.frame(x=c(689,655,655,689), y=c(322,322,208,208))
  pitch_mid_line <- data.frame(x=c(370,370), y=c(51,479))
  pitch_mid_circle <- circleFun(c(370,265),120,npoints = 100)
  pitch_left_gate <- data.frame(x=c(51,36,36,51), y=c(301,301,229,229))
  pitch_right_gate <- data.frame(x=c(689,704,704,689), y=c(301,301,229,229))
  
  
  for(i in 1:nrow(df)) {
    cur_player <- df[i,]
    player_name <- str_replace(cur_player$player_name, ' ', '_')
    
    df2 <- player_stats %>%
      filter(player_id == cur_player$player_id) %>% 
      select(player_stats_id=id, player_name)
    
    df3 <- inner_join(player_events, df2, by='player_stats_id') %>%
      mutate(event_type_short=substr(event_type, 1, 4))
    
    p <- ggplot(df3) + 
      geom_point(aes(x=x1, y=y1, group=1, shape=event_type), size=5, alpha=0.2) + 
      scale_shape_manual(values=1:length(unique(df3$event_type))) +
      facet_wrap(~event_type_short) +
      geom_path(data=pitch_boundary, aes(x, y,group=1)) +
      geom_path(data=pitch_left_outer_box, aes(x, y,group=1)) +
      geom_path(data=pitch_right_outer_box, aes(x, y,group=1)) +
      geom_path(data=pitch_left_inner_box, aes(x, y,group=1)) +
      geom_path(data=pitch_right_inner_box, aes(x, y,group=1)) +
      geom_line(data=pitch_mid_line, aes(x, y,group=1)) +
      geom_path(data=pitch_mid_circle, aes(x, y)) + 
      geom_path(data=pitch_left_gate, aes(x,y)) +
      geom_path(data=pitch_right_gate, aes(x,y)) +
      labs(title = player_name)
    
    ggsave(paste0(base_dir, "/",cur_player$cnt,"-",cur_player$player_id,"-",player_name,".png"), width=16.7, height=8) 
  }
}

############################################
#                Get data
############################################

conn <- dbConnect(dbDriver("SQLite"), dbname="fourfourtwo_primier_league_2015_2016.db")
player_stats <- dbGetQuery(conn, "select * from player_stats")
player_events <- dbGetQuery(conn, "select * from player_event")
matches <- dbGetQuery(conn, "select * from match")
dbDisconnect(conn)

matches$id <- as.integer(matches$id)
matches$season <- as.integer(matches$season)
matches$home_score <- as.integer(matches$home_score)
matches$away_score <- as.integer(matches$away_score)

player_stats$player_id <- as.integer(player_stats$player_id)
player_stats$match_id <- as.integer(player_stats$match_id)

player_events$event_half <- as.integer(player_events$event_half)
player_events$event_minute <- as.integer(player_events$event_minute)

players <- player_stats %>%
  group_by(player_id) %>%
  summarise(player_name=max(player_name)) %>%
  ungroup

# player events y-axis adjustment
player_events$y1 <- 530 - player_events$y1
player_events$y2 <- 530 - player_events$y2


############################################
#                Visualization
############################################
# number of appearence in a season greater than 36
df <- player_stats %>%
  group_by(player_id, player_name) %>%
  summarise(cnt=length(id)) %>%
  ungroup %>%
  filter(cnt > 36) %>%
  arrange(desc(cnt))

# number of appearence in a season greater than 36
team_name <- 'Arsenal'
match_list <- filter(matches, (home_team_name == team_name | away_team_name == team_name))$id
df <- player_stats %>%
  filter(match_id %in% match_list) %>%
  group_by(player_id, player_name) %>%
  summarise(cnt=length(id)) %>%
  ungroup %>%
  filter(cnt > 2) %>%
  arrange(desc(cnt))

draw_scatter_plot_of_player_activity(df, 'output-images/Arsenal')







